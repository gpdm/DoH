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

// init is the package int function
func init() {
	// random seed
	// we need this i.e. for doing randomized selection of DNS backend
	rand.Seed(time.Now().Unix())
}

// parseDNSQuestion inspects the DNS question from the payload packet,
// and implements telemetry logging.
func parseDNSQuestion(reqData []byte) (string, error) {
	// initialize the message parser
	var dnsParser dnsmessage.Parser

	// consume the dns message
	if _, err := dnsParser.Start(reqData); err != nil {
		return "", err
	}

	// parse the question
	for {
		q, err := dnsParser.Question()
		if err == dnsmessage.ErrSectionDone {
			break
		}
		if err != nil {
			return "", err
		}

		ConsoleLogger(LogDebug, q, false)
		ConsoleLogger(LogDebug, fmt.Sprintf("Lookup: %s, %s, %s\n", q.Name, q.Class, q.Type), false)

		// Telemetry: Logging DNS request type
		telemetryChannel <- TelemetryValues[q.Type.String()]
		ConsoleLogger(LogDebug, fmt.Sprintf("Logging DNS Telemetry for %s request.", q.Type), false)

		return "", nil
	}

	return "", nil
}

// parseDNSResponse inspects the DNS response, to return the lowest TTL,
// which then will be reflected to the HTTP response header
func parseDNSResponse(respData []byte) (uint32, error) {
	// initialize a DNS message
	var msg dnsmessage.Message
	// initialize a variable for the TTL
	var minimumTTL uint32 = 0

	// unpack the DNS packet
	err := msg.Unpack(respData)
	if err != nil {
		return 0, err
	}

	ConsoleLogger(LogDebug, fmt.Sprintf("DNS Response carries %d answer(s)\n", len(msg.Answers)), false)

	// parse the response
	for _, dnsRR := range msg.Answers {
		ConsoleLogger(LogDebug, fmt.Sprint("Response RR: ", dnsRR), false)
		ConsoleLogger(LogDebug, fmt.Sprintf("-> TTL is %d seconds\n", dnsRR.Header.TTL), false)

		// store minimum TTL if we have no value yet for the TTL
		// of if the previous value of the TTL is bigger than the current value
		if minimumTTL == 0 || minimumTTL > dnsRR.Header.TTL {
			minimumTTL = dnsRR.Header.TTL
		}
	}

	ConsoleLogger(LogDebug, fmt.Sprintf("Smallest TTL in response considered: %d", minimumTTL), false)

	// return a
	return minimumTTL, nil
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

	// because of our fixed-size buffer allocation above, we may end up with a zero-padded slice,
	// i.e. if we processed a standard 512-byte or less packet.
	// The payload must not be padded, so on return we
	// simply cut the slice to the number of bytes received.
	return response[:n], nil
}

// commonDNSRequestHandler is the shared backend routine, invoked from either
// the POST or GET frontend handlers.
// The routine consumes the http.ResponseWriter and dnsRequest,
// and passes the request to the question validator.
// If the DNS question is deemed valid, the query is passed over to the
// DNS server.
func commonDNSRequestHandler(w http.ResponseWriter, dnsRequest []byte) {
	// bail out if DNS request is smaller than 28 bytes
	if len(dnsRequest) < 28 {
		sendError(w, http.StatusBadRequest, "Malformed request: DNS payload is below treshold")
		return
	}

	// parse the DNS question
	if _, err := parseDNSQuestion(dnsRequest); err != nil {
		sendError(w, http.StatusBadRequest, fmt.Sprintf("Error in DNS question: %s", err))
		return
	}

	// FIXME: implement server-side cache, but honor ccNoCache
	dnsResponse, err := sendDNSRequest(dnsRequest)
	if err != nil {
		sendError(w, http.StatusBadRequest, fmt.Sprintf("Error during DNS resolution: %s", err))
		return
	}

	// parse DNS Response to get the minimum TTL
	minimumTTL, err := parseDNSResponse(dnsResponse)
	if err != nil {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("Error when parsing DNS response: %s", err))
		return
	}

	// FIXME: implement server-side cache, but honor ccNoStore
	cacheHandler(dnsResponse, minimumTTL)

	// return dns-message to client
	w.Header().Set("Content-Type", "application/dns-message")

	// reflect the minimum TTL into the response header
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age: %d", minimumTTL))

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
