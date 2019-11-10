/*
 * go DoH daemon
 *
 * This is a "DNS over HTTP" (DoH) redaemon solver written in Go.
 *
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 *
 * Provided to you under the terms of the BSD 3-Clause License
 *
 * Copyright (c) 2019. Gianpaolo Del Matto, https://github.com/gpdm, <delmatto _ at _ phunsites _ dot _ net>
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice, this
 *    list of conditions and the following disclaimer.
 *
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 * 3. Neither the name of the copyright holder nor the names of its
 *    contributors may be used to endorse or promote products derived from
 *    this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
 * SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
 * CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
 * OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 */

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/spf13/viper"

	// DoH lib
	goDoh "github.com/gpdm/DoH/go"
)

// setRuntimeConfig initializes the configuration defaults,
// parses configuration settings from the environment,
// potentially existing config files, and finally from CLI args
func setRuntimeConfig(logVerbose bool, logDebug bool, configFile string) {
	// set config defaults
	viper.SetDefault("global.listen", "")
	viper.SetDefault("global.loglevel", goDoh.LogNotice)
	viper.SetDefault("http.enable", false)
	viper.SetDefault("http.port", "80")
	viper.SetDefault("tls.port", "443")
	viper.SetDefault("tls.pkey", "./conf/private.key")
	viper.SetDefault("tls.cert", "./conf/public.crt")
	viper.SetDefault("dns.resolvers", []string{"localhost"})
	viper.SetDefault("redis.enable", false)
	viper.SetDefault("redis.addr", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.username", nil)
	viper.SetDefault("redis.password", nil)
	viper.SetDefault("influx.enable", false)
	viper.SetDefault("influx.url", nil)
	viper.SetDefault("influx.database", nil)
	viper.SetDefault("influx.username", nil)
	viper.SetDefault("influx.password", nil)

	// set default config file locations
	viper.SetConfigName("DoH")
	viper.AddConfigPath("/etc/DoH/")
	viper.AddConfigPath("./conf")

	// enable automatic config handling for env vars
	viper.AutomaticEnv()

	// perform config file/path location override magic, if given from CLI
	if configFile != "" {
		_, err := os.Stat(configFile)

		if !os.IsNotExist(err) {
			goDoh.ConsoleLogger(goDoh.LogNotice, fmt.Sprintf("using config file from: %s", configFile), false)
			viper.SetConfigFile(configFile)

		} else if os.IsNotExist(err) {
			goDoh.ConsoleLogger(goDoh.LogEmerg, fmt.Sprintf("error accessing '%s': %s", configFile, err), true) // fatal=true, will cause immediate exit
		}
	}

	// read configuration, and bail out on parser error
	if err := viper.ReadInConfig(); err != nil {
		// next line catches all errors, *except* file-not-found
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			goDoh.ConsoleLogger(goDoh.LogEmerg, err, true) // fatal=true, will cause immediate exit
		}
	}

	// permit overrides to log level from CLI.
	// this takes the highest precedence, and overrides
	// any potentially given config settings from ENV or config file.
	//
	// '-debug' takes precendence over '-verbose'
	if logVerbose { // overrides default log level
		viper.Set("global.loglevel", goDoh.LogInform)
	}
	if logDebug { // overrides previous log level
		viper.Set("global.loglevel", goDoh.LogDebug)
	}

	// print runtime configuration in verbose mode
	b, _ := json.MarshalIndent(viper.AllSettings(), "", "  ")
	goDoh.ConsoleLogger(goDoh.LogInform, fmt.Sprintf("Runtime Configuration dump:\n%s\n", string(b)), false)
}

// sanitizeRuntimeConfig applies some sanity checking against given
// configuration settings, and bails out if an error is encountered
func sanitizeRuntimeConfig() {
	// All calls to ConsoleLogger() set fatal=true,
	// which enforces an immediate abort from inside our logging routine.

	// bail out on missing cert/key files
	//
	if _, err := os.Stat(viper.GetString("tls.pkey")); err != nil {
		goDoh.ConsoleLogger(goDoh.LogEmerg, fmt.Sprintf("Error accessing TLS private key: %s", err), true)
	}
	if _, err := os.Stat(viper.GetString("tls.cert")); err != nil {
		goDoh.ConsoleLogger(goDoh.LogEmerg, fmt.Sprintf("Error accessing TLS certificate: %s", err), true)
	}

	// bail out on missing influxDB config
	//
	if viper.GetBool("influx.enable") && (viper.GetString("influx.url") == "" || viper.GetString("influx.username") == "" || viper.GetString("influx.password") == "" || viper.GetString("influx.database") == "") {
		goDoh.ConsoleLogger(goDoh.LogEmerg, "InfluxDB is enabled, but one or more required config values is not properly set.", true)
	}

	// bail out on missing influxDB config
	//
	if viper.GetBool("redis.enable") && (viper.GetString("redis.addr") == "" || viper.GetString("redis.port") == "" || viper.GetString("redis.username") == "" || viper.GetString("redis.password") == "") {
		goDoh.ConsoleLogger(goDoh.LogEmerg, "Redis is enabled, but one or more required config values is not properly set.", true)
	}

}

// main is our main routine
func main() {
	// wait group for go routines
	var wg sync.WaitGroup

	// our minimalistic CLI args support:
	// - accept optional override for config file
	// - also accept --verbose and --debug switches
	cfConfigFile := flag.String("configfile", "", "config file (optional)")
	rtVerbose := flag.Bool("verbose", false, "verbose mode")
	rtDebug := flag.Bool("debug", false, "debug mode")
	flag.Parse()

	// assemble our runtime configuration
	setRuntimeConfig(*rtVerbose, *rtDebug, *cfConfigFile)

	// sanitize our config
	sanitizeRuntimeConfig()

	// initialize influxDB telemetry collector
	TelemetryChannel := make(chan uint, 4096)
	go goDoh.TelemetryCollector(TelemetryChannel)

	// initialize HTTP service router
	router := goDoh.NewRouter(TelemetryChannel)

	// fire up optional HTTP-only server
	if viper.GetBool("http.enable") {
		wg.Add(1)
		go func() {
			goDoh.ConsoleLogger(goDoh.LogEmerg,
				http.ListenAndServe(fmt.Sprintf("%s:%s", viper.GetString("global.listen"), viper.GetString("http.port")), router),
				true)
		}()
		goDoh.ConsoleLogger(goDoh.LogNotice, "HTTP Server started", false)
	}

	// fire up TLS HTTP/2 server
	wg.Add(1)
	go func() {
		goDoh.ConsoleLogger(goDoh.LogEmerg,
			http.ListenAndServeTLS(fmt.Sprintf("%s:%s", viper.GetString("global.listen"), viper.GetString("tls.port")),
				viper.GetString("tls.cert"), viper.GetString("tls.pkey"), router),
			true)
	}()
	goDoh.ConsoleLogger(goDoh.LogNotice, "TLS HTTP Server started", false)

	// wait for all routines to complete
	// NOTE: will not currently happen, since we don't listen to any signals yet
	wg.Wait()

	os.Exit(0)
}
