/*
 * go DoH Daemon - Cache Control
 *
 * This is the "DNS over HTTP" (DoH) cache control package.
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

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// redisClient connects to a Redis instance and returns
// a connection handle
func redisClient() *redis.Pool {

	return &redis.Pool{
		// Maximum number of idle connections in the pool.
		MaxIdle: 80,
		// max number of connections
		MaxActive: 12000,
		// Dial is an application supplied function for creating and
		// configuring a connection.
		Dial: func() (redis.Conn, error) {
			logrus.Debugf("Connecting to Redis at %s", viper.GetString("redis.addr"))
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", viper.GetString("redis.addr"), viper.GetString("redis.port")))
			if err != nil {
				panic(err.Error())
			}
			if _, err := c.Do("AUTH", viper.GetString("redis.password")); err != nil {
				c.Close()
				return nil, err
			}
			return c, err
		},
	}

}

// redisAddToCache stores DNS responses as datasets to Redis.
// This function never fails, as errors are hidden from the caller.
// This allows the caller to continue independently from any potential
// error during the backend operation.
func redisAddToCache(dnsRequestID string, dnsResponse []byte, smallestTTL uint32) {
	// return if Redis is disabled
	if !viper.GetBool("redis.enable") {
		return
	}

	// connect to Redis
	// FIXME: connection pooling should propably be outside of this function
	// plus we should handle connection failures gracefully, i.e. to skip the cache
	// set/get in case of backend failure for better resilience (include redis' ping somewhere?)
	// FIXME return on redis unavailability
	pool := redisClient()
	c := pool.Get()
	defer c.Close()

	logrus.Debugf("Redis: storing response for %s (expire after %d seconds)", dnsRequestID, smallestTTL)

	// store object to redis
	_, err := c.Do("SET", dnsRequestID, dnsResponse)
	if err != nil {
		// handle cache-read errors gracefully, and return nil
		// so caller continues without cache result
		logrus.Debugf("Redis: error performing cache set: %s", err)
		return
	}

	// bind maximum object lifetime to max TTL value from the DNS response
	_, err = c.Do("EXPIRE", dnsRequestID, smallestTTL)
	if err != nil {
		// handle cache-read errors gracefully, and return nil
		// so caller continues without cache result
		logrus.Debugf("Redis: error performing cache expiration: %s", err)
		return
	}

	return
}

// redisGetFromCache retrieves potentially cached datasets from Redis,
// and converts them back to wire-format DNS response packets.
// This function never fails, as errors are hidden from the caller.
// This allows the caller to continue independently from any potential
// error during the backend operation.
func redisGetFromCache(dnsRequestID string) []byte {
	// return if Redis is disabled
	if !viper.GetBool("redis.enable") {
		return nil
	}

	// connect to Redis
	// FIXME: connection pooling should propably be outside of this function
	// plus we should handle connection failures gracefully, i.e. to skip the cache
	// set/get in case of backend failure for better resilience (include redis' ping somewhere?)
	// FIXME return on redis unavailability
	pool := redisClient()
	c := pool.Get()
	defer c.Close()

	logrus.Debugf("Redis: lookup for %s", dnsRequestID)

	// read object from Redis
	cachedDataset, err := c.Do("GET", dnsRequestID)
	if err != nil {
		// handle cache-read errors gracefully, and return nil
		// so caller continues without cache result
		logrus.Debugf("Redis: error performing cache lookup: %s", err)
		return nil
	}

	// return nil if no cached data was found, so caller
	// can continue without cache
	if cachedDataset == nil {
		logrus.Debugf("Redis: cache-miss, no data found")

		// Telemetry: Logging cache-miss
		telemetryChannel <- TelemetryValues["CacheMiss"]
		logrus.Debugf("Logging Redis Telemetry for cache-miss.")

		return nil
	}

	// convert redis dataset back to native byte-stream aka wire-format packet
	cachedDNSResponse, err := redis.Bytes(cachedDataset, err)
	if err != nil {
		// handle cache-conversion errors gracefully, and return nil
		// so caller continues without cache result
		logrus.Debugf("Redis: error performing cache conversion: %s", err)
		return nil
	}

	logrus.Debugf("Redis: cache-hit, retrieved %d bytes", len(cachedDNSResponse))

	// Telemetry: Logging cache-hit
	telemetryChannel <- TelemetryValues["CacheHit"]
	logrus.Debugf("Logging Redis Telemetry for cache-hit.")

	// return cached DNS response back to caller
	return cachedDNSResponse
}
