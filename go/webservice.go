/*
 * go DoH Daemon - HTTP API
 *
 * This is the "DNS over HTTP" (DoH) webservice package.
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
	"net/http"
)

// commonDNSRequestHandler is the shared backend routine, invoked from either
// the POST or GET frontend handlers.
// The routine consumes the http.ResponseWriter and dnsRequest,
// and passes the request to the question validator.
// If the DNS question is deemed valid, the query is passed over to the
// DNS server.
func commonDNSRequestHandler(w http.ResponseWriter, dnsRequest []byte) {
	// dnsRequestID is a Base64 generated from (DNS RR, Class, Type) from the DNS query
	// It servers as a lookup key in Redis to map cached requests/responses
	var dnsRequestID string
	// dnsResponse is the wire format byte stream received from either the Redis cache,
	// or - initially - from the upstream DNS servers
	var dnsResponse []byte
	// smallestTTL is used to control both Redis cache expiration,
	// and the http cache-control:max-age header, as mandated by RFC8484, Section 5.1
	var smallestTTL uint32

	// bail out if DNS request is smaller than 28 bytes
	if len(dnsRequest) < 28 {
		sendError(w, http.StatusBadRequest, "Malformed request: DNS payload is below treshold")
		return
	}

	// parse the DNS question
	dnsRequestID, err := parseDNSQuestion(dnsRequest)
	if err != nil {
		sendError(w, http.StatusBadRequest, fmt.Sprintf("Error in DNS question: %s", err))
		return
	}

	// perform cache lookup in redis
	if dnsResponse = redisGetFromCache(dnsRequestID); dnsResponse == nil {
		/*
		 * resolve DNS request if no cached data exists in redis
		 * (or when redis was disabled)
		 */

		dnsResponse, err = sendDNSRequest(dnsRequest)
		if err != nil {
			sendError(w, http.StatusBadRequest, fmt.Sprintf("Error during DNS resolution: %s", err))
			return
		}

		// parse DNS Response to get the minimum TTL
		smallestTTL, err = parseDNSResponse(dnsResponse)
		if err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("Error when parsing DNS response: %s", err))
			return
		}

		// store response to redis cache (unless redis is disabled)
		redisAddToCache(dnsRequestID, dnsResponse, smallestTTL)
	}

	// return dns-message to client
	w.Header().Set("Content-Type", "application/dns-message")

	// reflect the minimum TTL into the response header
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age: %d", smallestTTL))

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
	commonDNSRequestHandler(w, dnsRequest)

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
	commonDNSRequestHandler(w, dnsRequest)

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
