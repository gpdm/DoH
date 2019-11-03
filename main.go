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
	"log"
	"net/http"

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
	log.Printf("Server started")

	router := sw.NewRouter()

	//log.Fatal(http.ListenAndServe(":8081", router))
	log.Fatal(http.ListenAndServeTLS(":8080", "/Users/gianpaolo/Dropbox/Devel/lab.cer", "/Users/gianpaolo/Dropbox/Devel/lab.key", router))
}
