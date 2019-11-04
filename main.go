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
	"flag"
	"fmt"
	"log"
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
	// set default config file locations
	viper.SetConfigName("DoH")
	viper.AddConfigPath("/etc/DoH/")
	viper.AddConfigPath("./conf")
	// wait group for go routines
	var wg sync.WaitGroup

	// accept optional override for config file from CLI
	cfConfigFile := flag.String("configfile", "", "config file (optional)")
	flag.Parse()

	// perform config file/path location override magic, if given from CLI
	if *cfConfigFile != "" {
		_, err := os.Stat(*cfConfigFile)

		if !os.IsNotExist(err) {
			log.Printf("using config file from: %s", *cfConfigFile)
			viper.SetConfigFile(*cfConfigFile)

		} else if os.IsNotExist(err) {
			log.Printf("error accessing '%s': %s", *cfConfigFile, err)
			os.Exit(1)
		}
	}

	// read configuration, and bail out on parser error
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("%s", err)
			os.Exit(1)
		}
	}

	// bail out on missing cert/key files
	if _, err := os.Stat(viper.GetString("tls.pkey")); err != nil {
		log.Printf("Error accessing TLS private key: %s", err)
		os.Exit(1)
	}
	if _, err := os.Stat(viper.GetString("tls.cert")); err != nil {
		log.Printf("Error accessing TLS certificate: %s", err)
		os.Exit(1)
	}

	// initialize service router
	router := sw.NewRouter()

	// fire up optional http-only server
	if viper.GetBool("http.enable") {
		wg.Add(1)
		go func() {
			log.Fatal(http.ListenAndServe(
				fmt.Sprintf("%s:%s", viper.GetString("listen.address"), viper.GetString("http.port")), router))
		}()
		log.Printf("HTTP Server started")
	}

	// fire up TLS http/2 server
	wg.Add(1)
	go func() {
		log.Fatal(http.ListenAndServeTLS(
			fmt.Sprintf("%s:%s", viper.GetString("listen.address"), viper.GetString("tls.port")),
			viper.GetString("tls.cert"), viper.GetString("tls.pkey"), router))
	}()
	log.Printf("TLS Server started")

	// wait for all routines to complete
	wg.Wait()

	os.Exit(0)
}
