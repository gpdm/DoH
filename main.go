/*
 * Swagger DoH
 *
 * This is a DNS-over-HTTP (DoH) resolver written in Go.
 *
 * API version: 0.1
 * Contact: dev@phunsites.net
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

	// WARNING!
	// Change this to a fully-qualified import path
	// once you place this file into your project.
	// For example,
	//
	//    sw "github.com/myname/myrepo/go"
	//
	sw "./go"
)

func main() {
	// set config defaults
	viper.SetDefault("listen.address", "")
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
	viper.SetDefault("log.level", sw.LogNotice)
	// set default config file locations
	viper.SetConfigName("DoH")
	viper.AddConfigPath("/etc/DoH/")
	viper.AddConfigPath("./conf")
	// enable config handling for env vars
	viper.AutomaticEnv()
	// wait group for go routines
	var wg sync.WaitGroup

	// accept optional override for config file from CLI
	cfConfigFile := flag.String("configfile", "", "config file (optional)")
	rtVerbose := flag.Bool("verbose", false, "verbose mode")
	rtDebug := flag.Bool("debug", false, "debug mode")
	flag.Parse()

	// reflect CLI args to viper
	if *rtVerbose {
		viper.Set("log.level", sw.LogInform)
	}
	if *rtDebug {
		viper.Set("log.level", sw.LogDebug)
	}

	// perform config file/path location override magic, if given from CLI
	if *cfConfigFile != "" {
		_, err := os.Stat(*cfConfigFile)

		if !os.IsNotExist(err) {
			sw.ConsoleLogger(sw.LogNotice, fmt.Sprintf("using config file from: %s", *cfConfigFile), false)
			viper.SetConfigFile(*cfConfigFile)

		} else if os.IsNotExist(err) {
			sw.ConsoleLogger(sw.LogEmerg, fmt.Sprintf("error accessing '%s': %s", *cfConfigFile, err), true) // fatal=true, will cause immediate exit
		}
	}

	// read configuration, and bail out on parser error
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			sw.ConsoleLogger(sw.LogEmerg, err, true) // fatal=true, will cause immediate exit
		}
	}

	// bail out on missing cert/key files
	if _, err := os.Stat(viper.GetString("tls.pkey")); err != nil {
		sw.ConsoleLogger(sw.LogEmerg, fmt.Sprintf("Error accessing TLS private key: %s", err), true) // fatal=true, will cause immediate exit
	}
	if _, err := os.Stat(viper.GetString("tls.cert")); err != nil {
		sw.ConsoleLogger(sw.LogEmerg, fmt.Sprintf("Error accessing TLS certificate: %s", err), true) // fatal=true, will cause immediate exit
	}

	// FIXME
	// validate influxDB config
	// --missing-stuff-goes-here--

	// FIXME
	// validate redis config
	// --missing-stuff-goes-here--

	// print runtime configuration in verbose mode
	b, _ := json.MarshalIndent(viper.AllSettings(), "", "  ")
	sw.ConsoleLogger(sw.LogInform, fmt.Sprintf("Runtime Configuration dump:\n%s\n", string(b)), false)

	// initialize influxDB telemetry collector
	TelemetryChannel := make(chan uint, 4096)
	go sw.TelemetryCollector(TelemetryChannel)

	// initialize HTTP service router
	router := sw.NewRouter(TelemetryChannel)

	// fire up optional HTTP-only server
	if viper.GetBool("http.enable") {
		wg.Add(1)
		go func() {
			sw.ConsoleLogger(sw.LogEmerg,
				http.ListenAndServe(fmt.Sprintf("%s:%s", viper.GetString("listen.address"), viper.GetString("http.port")), router),
				true)
		}()
		sw.ConsoleLogger(sw.LogNotice, "HTTP Server started", false)
	}

	// fire up TLS HTTP/2 server
	wg.Add(1)
	go func() {
		sw.ConsoleLogger(sw.LogEmerg,
			http.ListenAndServeTLS(fmt.Sprintf("%s:%s", viper.GetString("listen.address"), viper.GetString("tls.port")),
				viper.GetString("tls.cert"), viper.GetString("tls.pkey"), router),
			true)
	}()
	sw.ConsoleLogger(sw.LogNotice, "TLS HTTP Server started", false)

	// wait for all routines to complete
	wg.Wait()

	os.Exit(0)

}
