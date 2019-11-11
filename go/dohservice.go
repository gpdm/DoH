/*
 * go DoH Daemon - HTTP API
 *
 * This is the "DNS over HTTP" (DoH) API functions library.
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

package dohservice

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/net/dns/dnsmessage"
)

// parseDNSQuestion inspects the DNS question from the payload packet,
// and implements some minimum logging output.
func parseDNSQuestion(reqData []byte) error {
	// initialize the message parser
	var dnsParser dnsmessage.Parser

	// consume the dns message
	if _, err := dnsParser.Start(reqData); err != nil {
		return err
	}

	// parse the question
	for {
		q, err := dnsParser.Question()
		if err == dnsmessage.ErrSectionDone {
			break
		}
		if err != nil {
			return err
		}

		ConsoleLogger(LogDebug, fmt.Sprintf("Lookup: %s, %s, %s\n", q.Name, q.Class, q.Type), false)

		// Telemetry: Logging DNS request type
		telemetryChannel <- TelemetryValues[q.Type.String()]
		ConsoleLogger(LogDebug, fmt.Sprintf("Logging DNS Telemetry for %s request.", q.Type), false)
	}

	return nil
}

func init() {
	// Only seed once.
	rand.Seed(time.Now().Unix())
}

/*
 * sendDNSRequest()
 *
 * send a DNS request to the resolver and return it's response
 */
func sendDNSRequest(request []byte) ([]byte, error) {
	// select a random DNS resolver.
	dnsResolver := viper.GetStringSlice("dns.resolvers")[rand.Intn(len(viper.GetStringSlice("dns.resolvers")))]

	// open UDP connection to DNS resolver
	udpConn, err := net.Dial("udp", fmt.Sprintf("%s:53", dnsResolver))
	if err != nil {
		return nil, err
	}
	defer udpConn.Close()
	timeout := time.Now().Add(time.Second * 30) // TODO: Make this configurable.
	if err := udpConn.SetDeadline(timeout); err != nil {
		return nil, fmt.Errorf("could not set deadline on udp conn: %w", err)
	}

	// send DNS request to resolver
	if _, err := udpConn.Write(request); err != nil {
		return nil, fmt.Errorf("could not send DNS request upstream: %w", err)
	}

	// Traditional DNS is limited to 512 bytes, while EDNS supports up to 2K.
	// We'll go with a 2K buffer.
	response := make([]byte, 2048)
	n, err := udpConn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("could not receive DNS response from upstream: %w", err)
	}

	return response[:n], nil // Cut the slice to the number of bytes received.
}

// commonDNSRequestHandler is the shared backend routine, invoked from either
// the POST or GET frontend handlers.
// The routine consumes the http.ResponseWrite, http.Request and dnsRequest,
// and passes the request to the question validator.
// If the DNS question is deemed valid, the query is passed over to the
// DNS server.
func commonDNSRequestHandler(w http.ResponseWriter, r http.Request, dnsRequest []byte) {
	var ccNoCache = false // default for Cache-Control: NoCache is FALSE (means: reply from cache)
	var ccNoStore = false // default for Cache-Control: NoStore is FALSE (means: store to cache)
	_ = ccNoCache         // FIXME: backend code for ccNoCache not there yet. Silence compiler warning using this directive

	// bail out if DNS request is smaller than 28 bytes
	if len(dnsRequest) < 28 {
		sendError(w, http.StatusBadRequest, "Malformed request: DNS payload is below treshold")
		return
	}

	// check Cache-Control request headers
	for _, value := range r.Header["Cache-Control"] {
		if value == "no-cache" {
			ConsoleLogger(LogDebug, "Client requested Cache-Control: no-cache", false)
			ccNoCache = true // client instructed to not response from server-side cache
		}
		if value == "no-store" {
			ConsoleLogger(LogDebug, "Client requested Cache-Control: no-store", false)
			ccNoStore = true // client instructed to not store to server-side cache
		}
	}

	// parse the DNS question
	if err := parseDNSQuestion(dnsRequest); err != nil {
		sendError(w, http.StatusBadRequest, fmt.Sprintf("Error in DNS question: %s", err))
		return
	}

	// FIXME: implement server-side cache, but honor ccNoCache
	dnsResponse, err := sendDNSRequest(dnsRequest)
	if err != nil {
		sendError(w, http.StatusBadRequest, fmt.Sprintf("Error during DNS resolution: %s", err))
		return
	}
	// FIXME: implement server-side cache, but honor ccNoStore

	// parse DNS Response
	/*
		FIXME: not implemented yet in backend code
	*/
	//parseDNSResponse(dnsResponse)

	// return dns-message to client
	w.Header().Set("Content-Type", "application/dns-message")

	// FIXME this is actually wrong
	// should implement DNS payload parser, and extract the proper TTL
	// for any RR type available. In case of SOA records, the MIN-TTL must be used instead.
	// honor ccNoStore flag: set max-age:0 to indicate we did our best to not cache
	if ccNoStore == true {
		w.Header().Set("Cache-Control", "max-age: 0")
	} else {
		// FIXME: this should actually be the *lowest* TTL from the DNS response
		// we need a DNS response parser to handle this
		w.Header().Set("Cache-Control", "max-age: 15")
	}

	// conclude with OK status code and return the dns payload
	w.WriteHeader(http.StatusOK)
	w.Write(dnsResponse)
}

// DNSQueryGet is the HTTP GET request handler, which performs
// minimum upfront validation, before passing the request over
// to the shared backend routine
func DNSQueryGet(w http.ResponseWriter, r *http.Request) {

	// validate 'dns' attribute from GET request arguments
	argDNS, ok := r.URL.Query()["dns"]
	if !ok || len(argDNS[0]) < 1 || len(argDNS) > 1 {
		// bail out if 'dns' is omitted, zero-length or occurs multiple times
		sendError(w, http.StatusBadRequest, "Mandatory 'dns' request parameter is either not set, empty, or defined multiple times")
		return
	}

	// decode the DNS request
	// NOTE: this uses RFC4648 URL encoding *without* padding
	dnsRequest, err := base64.RawURLEncoding.DecodeString(argDNS[0])
	if err != nil {
		sendError(w, http.StatusBadRequest, fmt.Sprintf("Error decoding DNS request data from Base64: %s", err))
		return
	}

	// pass DNS request to request handler
	commonDNSRequestHandler(w, *r, dnsRequest)

	return
}

// DNSQueryPost is the HTTP POST request handler, which performs
// minimum upfront validation, before passing the request over
// to the shared backend routine
func DNSQueryPost(w http.ResponseWriter, r *http.Request) {

	// bail out on unsupported content-type
	if r.Header.Get("Content-Type") != "application/dns-message" {
		sendError(w, http.StatusUnsupportedMediaType, "unsupported or missing Content-Type")
		return
	}

	// some minimal sanity checking on the message body
	if r.Body == nil {
		sendError(w, http.StatusBadRequest, "Missing body payload")
		return
	}

	// get DNS request from message body
	dnsRequest, _ := ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewBuffer(dnsRequest))

	// bail out if 'body' is zero-length
	if len(dnsRequest) == 0 {
		// bail out if 'body' is zero-length
		sendError(w, http.StatusBadRequest, "Missing dns message payload")
		return
	}

	// pass DNS request to request handler
	commonDNSRequestHandler(w, *r, dnsRequest)

	return
}

// rootIndex is the HTTP request handler, which is called
// upon accessing the web service index / document root.
// This function returns a generic message to the client.
func rootIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "DoH Server")
}

// status is the Status Handler, which indicates if the service is healthly
// This is only meant as a public "ping" method, nothing more, nothing less.
func status(w http.ResponseWriter, r *http.Request) {
	// this section needs improving
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Server is running")
}
