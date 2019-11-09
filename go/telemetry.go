/*
 * DoH Service - Telemetry Sender
 *
 * This is the telemetry sender, which sends statistical information to InfluxDB
 *
 * Contact: dev@phunsites.net
 */

package dohservice

import (
	"fmt"
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/spf13/viper"
)

const TelemetryHttpRequestTypeGet uint = 0b00000001
const TelemetryHttpRequestTypePost uint = 0b00000010
const TelemetryDnsRequestTypeAny uint = 0b00000000
const TelemetryDnsRequestTypeA uint = 0b00000011
const TelemetryDnsRequestTypeAAAA uint = 0b00000100

// TelemetryValues serves as a lookup table to map given keywords to a binary type.
// The binary type will be reflected over the IPC channel,
// in order to not fummel around with string literals
//
// TelemetryValues is a public map, so external functions can make use of this.
var TelemetryValues = map[string]uint{
	"POST":     TelemetryHttpRequestTypePost,
	"GET":      TelemetryHttpRequestTypeGet,
	"TypeANY":  TelemetryDnsRequestTypeAny,
	"TypeA":    TelemetryDnsRequestTypeA,
	"TypeAAAA": TelemetryDnsRequestTypeAAAA,
}

// telemetryData maps the binary values back onto a more useful map,
// we is used to bring the data into contect and track the statistics
//
// telemetryData is a private map.
var telemetryData = map[uint]map[string]interface{}{
	TelemetryHttpRequestTypePost: {
		"RequestCategory": "HTTP",
		"RequestType":     "POST",
		"RequestCounter":  0,
	},
	TelemetryHttpRequestTypeGet: {
		"RequestCategory": "HTTP",
		"RequestType":     "GET",
		"RequestCounter":  0,
	},
	TelemetryDnsRequestTypeAny: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeANY",
		"RequestCounter":  0,
	},
	TelemetryDnsRequestTypeA: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeA",
		"RequestCounter":  0,
	},
	TelemetryDnsRequestTypeAAAA: {
		"RequestCategory": "DNS",
		"RequestType":     "TypeAAAA",
		"RequestCounter":  0,
	},
}

// influxDBClient connects to an InfluxDB instance and returns
// a connection handle
func influxDBClient() client.Client {
	ConsoleLogger(LogDebug, fmt.Sprintf("Connecting to InfluxDB at %s", viper.GetString("influx.url")), false)
	influxConnection, connectionError := client.NewHTTPClient(client.HTTPConfig{
		Addr:     viper.GetString("influx.url"),
		Username: viper.GetString("influx.username"),
		Password: viper.GetString("influx.password"),
	})
	if connectionError != nil {
		ConsoleLogger(LogCrit, fmt.Sprintf("Error connecting to InfluxDB: %s", connectionError), true)
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
	// loop our statistics map
	for _requestType, _ := range telemetryData {
		// reset counter
		telemetryData[_requestType]["RequestCounter"] = 0
	}
}

func sendMetrics(c client.Client) {
	bp, bpError := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  viper.GetString("influx.database"),
		Precision: "s",
	})
	if bpError != nil {
		ConsoleLogger(LogCrit, fmt.Sprintf("Error connecting to InfluxDB: %s", bpError), true)
	}

	//eventTime := time.Now().Add(time.Second * -20)

	httpPoint, httpPointError := client.NewPoint(
		"dohStatistics",
		map[string]string{ // tags
			"ServiceStats": "HTTP",
		},
		getCounters("HTTP"), // fields
		//eventTime.Add(time.Second*10),
		time.Now(),
	)
	if httpPointError != nil {
		ConsoleLogger(LogCrit, fmt.Sprintf("Error assembling report point: %s", httpPointError), true)
	}

	dnsPoint, dnsPointError := client.NewPoint(
		"dohStatistics",
		map[string]string{ // tags
			"ServiceStats": "DNS",
		},
		getCounters("DNS"), // fields
		//eventTime.Add(time.Second*10),
		time.Now(),
	)
	if dnsPointError != nil {
		ConsoleLogger(LogCrit, fmt.Sprintf("Error assembling report point: %s", dnsPointError), true)
	}

	bp.AddPoint(httpPoint)
	bp.AddPoint(dnsPoint)

	writeError := c.Write(bp)
	if writeError != nil {
		ConsoleLogger(LogCrit, fmt.Sprintf("Error writing to InfluxDB: %s", writeError), true)
	}
}

// TelemetryCollector receives information from other
// go routines and forwards them to InfluxDB
func TelemetryCollector(chanTelemetry chan uint) {
	// track when Telemetry was last comitted to InfluxDB
	var telemetryLastUpdate = time.Now().Unix()

	// Check if InfluxDB is disabled.
	//
	// If this is the case, divert into this specific event loop.
	// Since other go routines will still throw telemtry to the collector,
	// we need to consume the telemtry channel in order to prevent deadlocks.
	// The alternative would be to clutter the code with if-else's.
	// The expense for not doing this, is to do the extra-roundtrip to the
	// collector, and simply throw away the data.
	//
	if !viper.GetBool("influx.enable") {
		ConsoleLogger(LogInform, "InfluxDB Telemetry Forwarding is disabled", false)

		// stay in loop forever
		for {
			// discard telemetry data
			_ = <-chanTelemetry
			ConsoleLogger(LogDebug, "Received Telemetry was internally discarded", false)
		}
		// we never end up here since the loop has no break condition
	}

	// connect to InfluxDB
	c := influxDBClient()
	defer c.Close()

	// stay in loop forever
	for {
		// consume telemetry data
		// telemetry data will consist of a binary value
		receivedTelemetry := <-chanTelemetry
		ConsoleLogger(LogDebug, fmt.Sprintf("Received incoming telemetry: %s", telemetryData[receivedTelemetry]["RequestType"]), false)

		// telemetry counters use the telemetry's value as the key,
		// so we can just throw it in to the map in order to increment the counters
		// telemetryDataA[receivedTelemetry].RequestCounter = telemetryDataA[receivedTelemetry].RequestCounter
		telemetryData[receivedTelemetry]["RequestCounter"] = (telemetryData[receivedTelemetry]["RequestCounter"].(int)) + 1
		ConsoleLogger(LogDebug, fmt.Sprint("New Count for telementry: ", telemetryData), false)

		// send new aggregate telemetry information to InfluxDB
		// only every other second
		if time.Now().Unix() > telemetryLastUpdate+1 {
			ConsoleLogger(LogDebug, "InfluxDB: sending telemetry update", false)
			sendMetrics(c)

			// reset counters
			ConsoleLogger(LogDebug, "Resetting telemtry counters", false)
			resetCounters()

			// refresh last update timestamp
			telemetryLastUpdate = time.Now().Unix()
		}
	}
	// we never end up here since the loop has no break condition
}
