/*
 * go DoH Daemon - Telemetry Sender
 *
 * This is the telemetry sender, which sends statistical information to InfluxDB
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
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// telemetryChannel is globally registered into the package,
// so other functions can make use of it as well.
var telemetryChannel chan uint = nil

// TelemetryDNSRequestTypeALL is an arbitary type to track DNS ALL requests
const TelemetryDNSRequestTypeALL uint = 0b11111111

// TelemetryDNSRequestTypeA is an arbitary type to track DNS A requests
const TelemetryDNSRequestTypeA uint = 0b00000001

// TelemetryDNSRequestTypeAAAA is an arbitary type to track DNS AAAA requests
const TelemetryDNSRequestTypeAAAA uint = 0b00011100

// TelemetryDNSRequestTypeCNAME is an arbitary type to track DNS CNAME requests
const TelemetryDNSRequestTypeCNAME uint = 0b00000101

// TelemetryDNSRequestTypeHINFO is an arbitary type to track DNS HINFO requests
const TelemetryDNSRequestTypeHINFO uint = 0b00001101

// TelemetryDNSRequestTypeMINFO is an arbitary type to track DNS MINFO requests
const TelemetryDNSRequestTypeMINFO uint = 0b00001110

// TelemetryDNSRequestTypeMX is an arbitary type to track DNS MX requests
const TelemetryDNSRequestTypeMX uint = 0b00001111

// TelemetryDNSRequestTypeNS is an arbitary type to track DNS NS requests
const TelemetryDNSRequestTypeNS uint = 0b00000010

// TelemetryDNSRequestTypePTR is an arbitary type to track DNS PTR requests
const TelemetryDNSRequestTypePTR uint = 0b00001100

// TelemetryDNSRequestTypeSOA is an arbitary type to track DNS SOA requests
const TelemetryDNSRequestTypeSOA uint = 0b00000110

// TelemetryDNSRequestTypeSRV is an arbitary type to track DNS SRV requests
const TelemetryDNSRequestTypeSRV uint = 0b00100001

// TelemetryDNSRequestTypeTXT is an arbitary type to track DNS TXT requests
const TelemetryDNSRequestTypeTXT uint = 0b00010000

// TelemetryDNSRequestTypeWKS is an arbitary type to track DNS WKS requests
const TelemetryDNSRequestTypeWKS uint = 0b00001011

// TelemetryHTTPRequestTypeGet is an arbitary type to track HTTP GET requests
const TelemetryHTTPRequestTypeGet uint = 0b0000001000000000

// TelemetryHTTPRequestTypePost is an arbitary type to track HTTP POST requests
const TelemetryHTTPRequestTypePost uint = 0b0000001000000001

// TelemetryRedisCacheHit is an arbitary type to track Redis cache hits
const TelemetryRedisCacheHit uint = 0b0000001000000010

// TelemetryRedisCacheMiss is an arbitary type to track Redis cache misses
const TelemetryRedisCacheMiss uint = 0b0000001000000100

// TelemetryKeepAlive is a generic type to track internal keep-alive
const TelemetryKeepAlive uint = 0b1111111111111111

// TelemetryValues serves as a lookup table to map given keywords to a binary type.
// The binary type will be reflected over the IPC channel,
// in order to not fummel around with string literals
//
// TelemetryValues is a public map, so external functions can make use of this.
var TelemetryValues = map[string]uint{
	"POST":      TelemetryHTTPRequestTypePost,
	"GET":       TelemetryHTTPRequestTypeGet,
	"TypeANY":   TelemetryDNSRequestTypeALL,
	"TypeA":     TelemetryDNSRequestTypeA,
	"TypeAAAA":  TelemetryDNSRequestTypeAAAA,
	"TypeHINFO": TelemetryDNSRequestTypeHINFO,
	"TypeMINFO": TelemetryDNSRequestTypeMINFO,
	"TypeMX":    TelemetryDNSRequestTypeMX,
	"TypeNS":    TelemetryDNSRequestTypeNS,
	"TypePTR":   TelemetryDNSRequestTypePTR,
	"TypeSOA":   TelemetryDNSRequestTypeSOA,
	"TypeSRV":   TelemetryDNSRequestTypeSRV,
	"TypeTXT":   TelemetryDNSRequestTypeTXT,
	"TypeWKS":   TelemetryDNSRequestTypeWKS,
	"CacheHit":  TelemetryRedisCacheHit,
	"CacheMiss": TelemetryRedisCacheMiss,
	"KeepAlive": TelemetryKeepAlive,
}

// telemetryData maps the binary values back onto a more useful map,
// we is used to bring the data into contect and track the statistics
//
// telemetryData is a private map.
var telemetryData = map[uint]map[string]interface{}{
	TelemetryHTTPRequestTypePost: {
		"RequestCategory": "HTTP",
		"RequestType":     "POST",
		"RequestCounter":  0,
	},
	TelemetryHTTPRequestTypeGet: {
		"RequestCategory": "HTTP",
		"RequestType":     "GET",
		"RequestCounter":  0,
	},
	TelemetryDNSRequestTypeALL: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeALL",
		"RequestCounter":  0,
	},
	TelemetryDNSRequestTypeA: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeA",
		"RequestCounter":  0,
	},
	TelemetryDNSRequestTypeAAAA: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeAAAA",
		"RequestCounter":  0,
	},
	TelemetryDNSRequestTypeHINFO: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeHINFO",
		"RequestCounter":  0,
	},
	TelemetryDNSRequestTypeMINFO: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeMINFO",
		"RequestCounter":  0,
	},
	TelemetryDNSRequestTypeMX: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeMX",
		"RequestCounter":  0,
	},
	TelemetryDNSRequestTypeNS: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeNS",
		"RequestCounter":  0,
	},
	TelemetryDNSRequestTypePTR: {
		"RequestCategory": "DNS",
		"RequestType":     "TypePTR",
		"RequestCounter":  0,
	},
	TelemetryDNSRequestTypeSOA: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeSOA",
		"RequestCounter":  0,
	},
	TelemetryDNSRequestTypeSRV: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeSRV",
		"RequestCounter":  0,
	},
	TelemetryDNSRequestTypeTXT: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeTXT",
		"RequestCounter":  0,
	},
	TelemetryDNSRequestTypeWKS: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeWKS",
		"RequestCounter":  0,
	},
	TelemetryRedisCacheHit: {
		"RequestCategory": "Redis",
		"RequestType":     "CacheHit",
		"RequestCounter":  0,
	},
	TelemetryRedisCacheMiss: {
		"RequestCategory": "Redis",
		"RequestType":     "CacheMiss",
		"RequestCounter":  0,
	},
	TelemetryKeepAlive: {
		"RequestCategory": "KeepAlive",
		"RequestType":     "KeepAlive",
		"RequestCounter":  0,
	},
}

// influxDBClient connects to an InfluxDB instance and returns
// a connection handle
func influxDBClient() client.Client {
	logrus.Debugf("Connecting to InfluxDB at %s", viper.GetString("influx.url"))
	influxConnection, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     viper.GetString("influx.url"),
		Username: viper.GetString("influx.username"),
		Password: viper.GetString("influx.password"),
	})
	if err != nil {
		logrus.Fatalf("Error connecting to InfluxDB: %s", err)
	}
	return influxConnection
}

// getCounters parses our telemetry statistics
// and looks for a given request category, returning
// a fields map matching all applicable stats counters
func getCounters(neededRequestCategory string) map[string]interface{} {
	// a prototype fields map to which we export our stats counters
	influxFields := map[string]interface{}{}

	// loop our statistics map
	for _, _requestData := range telemetryData {
		// skip if records is not matching our request category
		if _requestData["RequestCategory"] != neededRequestCategory {
			continue
		}

		// stringify retrieved request-type as it's of type interface{}
		// and assign the counter
		influxFields[_requestData["RequestType"].(string)] = _requestData["RequestCounter"]
	}

	return influxFields
}

// resetCounters parses our telemetry statistics
// and resets all current counts to zero
func resetCounters() {
	logrus.Debugf("Resetting telemetry counters")

	// loop our statistics map
	for _requestType := range telemetryData {
		// reset counter
		telemetryData[_requestType]["RequestCounter"] = 0
	}
}

// sendMetrics parses the telemetry information out into
// datastructures suitable to for InfluxDB, to which it is sent.
func sendMetrics(c client.Client) bool {
	logrus.Debugf("InfluxDB: sending telemetry update")

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  viper.GetString("influx.database"),
		Precision: "s",
	})
	if err != nil {
		logrus.Fatalf("Error connecting to InfluxDB: %s", err)
		return false
	}

	// time series for HTTP requests
	httpPoint, err := client.NewPoint(
		"dohStatistics",
		map[string]string{ // tags
			"ServiceStats": "HTTP",
		},
		getCounters("HTTP"), // fields
		time.Now(),
	)
	if err != nil {
		logrus.Fatalf("Error assembling report point: %s", err)
		return false
	}

	// time series for DNS
	dnsPoint, err := client.NewPoint(
		"dohStatistics",
		map[string]string{ // tags
			"ServiceStats": "DNS",
		},
		getCounters("DNS"), // fields
		time.Now(),
	)
	if err != nil {
		logrus.Fatalf("Error assembling report point: %s", err)
		return false
	}

	// time series for Redis
	redisPoint, err := client.NewPoint(
		"dohStatistics",
		map[string]string{ // tags
			"ServiceStats": "Redis",
		},
		getCounters("Redis"), // fields
		time.Now(),
	)
	if err != nil {
		logrus.Fatalf("Error assembling report point: %s", err)
		return false
	}

	// push all time series points
	bp.AddPoint(httpPoint)
	bp.AddPoint(dnsPoint)
	bp.AddPoint(redisPoint)

	if err := c.Write(bp); err != nil {
		logrus.Fatalf("Error writing to InfluxDB: %s", err)
		return false
	}

	// reset counters on successful commit
	resetCounters()

	return true
}

// TelemetryCollector receives information from other
// go routines and forwards them to InfluxDB
func TelemetryCollector(chanTelemetry chan uint) {
	// track when Telemetry was last comitted to InfluxDB
	var tick = time.Tick(time.Second * 60) // Keepalive ticker.
	var tock = time.Tick(time.Second * 2)  // Aggregation interval.

	// register global telemetry channel
	telemetryChannel = chanTelemetry

	// Check if InfluxDB is disabled.
	//
	// If this is the case, divert into this specific event loop.
	// Since other go routines will still throw telemetry to the collector,
	// we need to consume the telemetry channel in order to prevent deadlocks.
	// The alternative would be to clutter the code with if-else's.
	// The expense for not doing this, is to do the extra-roundtrip to the
	// collector, and simply throw away the data.
	influxEnabled := viper.GetBool("influx.enable")

	// connect to InfluxDB
	var c client.Client
	if influxEnabled {
		c = influxDBClient()
		defer c.Close()
	}

	// stay in loop forever
	for {
		select {
		case <-tick:
			// We got a keepalive.
			chanTelemetry <- TelemetryValues["KeepAlive"]
			logrus.Debugf("Logging Telemetry keep-alive.")
		case receivedTelemetry := <-chanTelemetry:
			// Only send to influx if it's enabled.
			if !influxEnabled {
				logrus.Debugf("Received Telemetry was internally discarded.")
				continue
			}

			// Consume telemetry data.
			// Telemetry data will consist of a binary value.
			logrus.Debugf("Received incoming telemetry: %s", telemetryData[receivedTelemetry]["RequestType"])

			// telemetry counters use the telemetry's value as the key,
			// so we can just throw it in to the map in order to increment the counters
			telemetryData[receivedTelemetry]["RequestCounter"] = (telemetryData[receivedTelemetry]["RequestCounter"].(int)) + 1
			logrus.Debugf("New Count for telementry: ", telemetryData)

			// send new aggregate telemetry information to InfluxDB
			// only every other second
			select {
			case <-tock:
				// It's time to send a telemetry update.
				sendMetrics(c)

			default:
				// Nope, not yet.
			}
		}
	}
	// we never end up here since the loop has no break condition
}
