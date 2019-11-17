/*
 * go DoH Daemon - DNS recurser
 *
 * This is the "DNS over HTTP" (DoH) DNS recurser package.
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
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
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
		if err != nil {
			return "", err
		}

		ConsoleLogger(LogDebug, fmt.Sprintf("Lookup: %s, %s, %s\n", q.Name, q.Class, q.Type), false)

		// Telemetry: Logging DNS request type
		telemetryChannel <- TelemetryValues[q.Type.String()]
		ConsoleLogger(LogDebug, fmt.Sprintf("Logging DNS Telemetry for %s request.", q.Type), false)

		// return a Base64 encoded string generated from (DNS RR, Class and Type)
		// this string will be used to perform cache set/get actions
		return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s:%s", q.Name, q.Class, q.Type))), nil
	}
}

// parseDNSResponse inspects the DNS response, to return the lowest TTL,
// which then will be reflected to the HTTP response header
func parseDNSResponse(respData []byte) (uint32, error) {
	// initialize a DNS message
	var msg dnsmessage.Message
	// smallestTTL holds the smallest TTL found in any response.
	// As per RFC8484, Section 5.1, the smallest <of-many> TTLs must
	// be used, i.e. to apply for cache-control:max-age
	// Likewise, the same value is used to steer the Radis cache behaviour
	var smallestTTL uint32 = 0

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
		if smallestTTL == 0 || smallestTTL >= dnsRR.Header.TTL {
			smallestTTL = dnsRR.Header.TTL
		}
	}

	ConsoleLogger(LogDebug, fmt.Sprintf("Smallest TTL in response considered: %d", smallestTTL), false)

	// return a
	return smallestTTL, nil
}

/*
 * sendDNSRequest()
 *
 * send a DNS request to the resolver and return it's response
 */
func sendDNSRequest(request []byte) ([]byte, error) {
	// select a random DNS resolver.
	// FIXME we should have a global probing mechanism, which marks/unmarks servers as alive/dead
	// and then select the backends only from the number of servers alive
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
