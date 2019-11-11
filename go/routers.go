/*
 * go DoH Daemon - HTTP Router
 *
 * This is the collection for the HTTP request router
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
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type routes []route

// routers defines set of HTTP handler routes
var routers = routes{
	route{
		"Index",
		"GET",
		"/",
		rootIndex,
	},

	route{
		"Status",
		"GET",
		"/status",
		status,
	},

	route{
		"DNSQueryGet",
		strings.ToUpper("Get"),
		"/dns-query",
		DNSQueryGet,
	},

	route{
		"DNSQueryPost",
		strings.ToUpper("Post"),
		"/dns-query",
		DNSQueryPost,
	},
}

// NewRouter initializes an HTTP multiplexer for the webservice
func NewRouter(chanTelemetry chan uint) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routers {
		var handler http.Handler
		handler = httpHandler(route.HandlerFunc, route.Name, chanTelemetry)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)

		ConsoleLogger(LogInform, fmt.Sprintf("Registered HTTP handler: method=%s, path=%s", route.Method, route.Pattern), false)
	}

	return router
}

// httpHandler wraps the http request handler and logging routine.
func httpHandler(inner http.Handler, name string, chanTelemetry chan uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// add some extra verbosity before we handle the request
		ConsoleLogger(LogDebug, fmt.Sprintf("Client Requested URL: %s", r.URL), false)
		ConsoleLogger(LogDebug, fmt.Sprintf("Client Request Headers: %s", r.Header), false)

		// Telemetry: Logging HTTP request type
		chanTelemetry <- TelemetryValues[r.Method]
		ConsoleLogger(LogDebug, fmt.Sprintf("Logging HTTP Telemetry for %s request.", r.Method), false)

		// serve the HTTP request
		inner.ServeHTTP(w, r)

		// Logging HTTP request in verbose mode
		ConsoleLogger(LogInform, fmt.Sprintf(
			"%s %s %s %s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		), false)
	})
}

// sendError is a helper to construct meaningful error messages
// returned to the client
func sendError(w http.ResponseWriter, httpStatusCode int, errorMessage string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(httpStatusCode)
	fmt.Fprintf(w, errorMessage)
	ConsoleLogger(LogDebug, errorMessage, false)

	return
}
