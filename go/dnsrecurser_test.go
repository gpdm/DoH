/*
 * go DoH Daemon - DNS recurser test suite
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
	"io/ioutil"
	"testing"
)

// TestSendDNSRequestWithoutActiveResolvers tests if we get a proper error
// in case no DNS resolvers are available.
func TestSendDNSRequestWithoutActiveResolvers(t *testing.T) {

	// clear all active resolvers
	ActiveDNSResolvers = nil

	// load data file
	request, err := ioutil.ReadFile("../testdata/A_www.example.com.bin")
	if err != nil {
		t.Errorf("Reading data file failed with error: %v", err)
		return
	}

	// send request
	response, err := sendDNSRequest(request)

	// test succeeds if we get "no resolvers available" error
	if err.Error() == "No active DNS resolvers available (all targets are offline)" {
		t.Logf("sendDNSRequest() returned *expected* error: %v", err)
		return
	}

	// test fails if we get any other kind of error
	if err != nil {
		t.Errorf("sendDNSRequest() returned unexpected error: %v", err)
		return
	}

	// test fails if we didn't get an error, but an actual response
	t.Errorf("sendDNSRequest() returned no error at all, but an unexpected response: %v", response)
}

// TestSendDNSRequestWithRandomResolverSelection checks random resolver selection
// by iterating over a list of valid backends several times
func TestSendDNSRequestWithRandomResolverSelection(t *testing.T) {

	// assign list of valid resolvers
	ActiveDNSResolvers = collectionOfValidResolvers

	// load data file
	request, err := ioutil.ReadFile("../testdata/A_www.example.com.bin")
	if err != nil {
		t.Errorf("Reading data file failed with error: %v", err)
		return
	}

	// loop 10 times to send request to random resolvers
	for i := 1; i <= 10; i++ {
		response, err := sendDNSRequest(request)

		if err != nil {
			t.Errorf("sendDNSRequest() failed with error: %v", err)
			return
		}

		t.Logf("sendDNSRequest() succeeded with response: %v", response)
	}
}

// TestSendDNSRequestUDP checks UDP Resolver
func TestSendDNSRequestUDP(t *testing.T) {

	// assign list of valid resolvers
	ActiveDNSResolvers = []DNSResolver{validUDPResolverGoogle}

	// load data file
	request, err := ioutil.ReadFile("../testdata/A_www.example.com.bin")
	if err != nil {
		t.Errorf("Reading data file failed with error: %v", err)
		return
	}

	response, err := sendDNSRequest(request)

	if err != nil {
		t.Errorf("sendDNSRequest() failed with error: %v", err)
		return
	}

	t.Logf("sendDNSRequest() succeeded with response: %v", response)
}

// TestSendDNSRequestDoHPost checks DoH Resolver through HTTP-POST
func TestSendDNSRequestDoHPost(t *testing.T) {

	// assign list of valid resolvers
	ActiveDNSResolvers = []DNSResolver{validDoHResolverCloudFlarePost}

	// load data file
	request, err := ioutil.ReadFile("../testdata/A_www.example.com.bin")
	if err != nil {
		t.Errorf("Reading data file failed with error: %v", err)
		return
	}

	response, err := sendDNSRequest(request)

	if err != nil {
		t.Errorf("sendDNSRequest() failed with error: %v", err)
		return
	}

	t.Logf("sendDNSRequest() succeeded with response: %v", response)
}

// TestSendDNSRequestDoHGet checks DoH Resolver through HTTP-GET
func TestSendDNSRequestDoHGet(t *testing.T) {

	// assign list of valid resolvers
	ActiveDNSResolvers = []DNSResolver{validDoHResolverCloudFlareGet}

	// load data file
	request, err := ioutil.ReadFile("../testdata/A_www.example.com.bin")
	if err != nil {
		t.Errorf("Reading data file failed with error: %v", err)
		return
	}

	response, err := sendDNSRequest(request)

	if err != nil {
		t.Errorf("sendDNSRequest() failed with error: %v", err)
		return
	}

	t.Logf("sendDNSRequest() succeeded with response: %v", response)
}
