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
	"encoding/json"
	"fmt"
	"log"

	"github.com/gomodule/redigo/redis"
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
			ConsoleLogger(LogDebug, fmt.Sprintf("Connecting to Redis at %s", viper.GetString("redis.addr")), false)
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

type cachedDNSRequest struct {
	requestData  string
	responseData []byte
	cachedTTL    uint32
}

func cacheHandler(dnsResponse []byte, minimumTTL uint32) {
	if !viper.GetBool("redis.enable") {
		return
	}

	// connect to InfluxDB
	pool := redisClient()
	c := pool.Get()
	defer c.Close()

	//const objectPrefix string = "user:"

	usr := cachedDNSRequest{
		requestData:  "test",
		responseData: dnsResponse,
		cachedTTL:    minimumTTL,
	}

	// serialize User object to JSON
	json, err := json.Marshal(usr)
	if err != nil {
		log.Println(err)
		//return err
	}

	// SET object
	_, err = c.Do("SET", usr.requestData, json)
	if err != nil {
		log.Println(err)
		//return err
	}

	return
}
