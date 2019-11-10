/*
 * go DoH Daemon - Logging Functions
 *
 * This is the logging collection, which provides some abstracts for log handling
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
	"log"

	"github.com/spf13/viper"
)

// LogEmerg is a Syslog-type priority for Emergency messages
const LogEmerg uint = 0b00000000

// LogAlert is a Syslog-type priority for Alert messages
const LogAlert uint = 0b00000001

// LogCrit is a Syslog-type priority for Critical messages
const LogCrit uint = 0b00000010

// LogErr is a Syslog-type priority for Error messages
const LogErr uint = 0b00000011

// LogWarn is a Syslog-type priority for Warning messages
const LogWarn uint = 0b00000100

// LogNotice is a Syslog-type priority for Notice messages
const LogNotice uint = 0b00000101

// LogInform is a Syslog-type priority for Informational messages
const LogInform uint = 0b00000110

// LogDebug is a Syslog-type priority for Debug messages
const LogDebug uint = 0b00000111

// ConsoleLogger is a wrapper to the standard logging function.
// In order to not clutter the code all over with if-else's to cope
// with special logging needs (i.e. print logs in verbose and/or debug mode,
// but not otherwise), this function takes care to do this properly.
func ConsoleLogger(logPriority uint, logMessage interface{}, fatal bool) {

	// skip logging output if priority does not match configured log level
	if logPriority > viper.GetUint("log.level") {
		return
	}

	// check if a fatal error must be reported
	if fatal {
		// log the message, calling .Fatal, which will implicitly call os.Exit()
		log.Fatal(logMessage)
	} else {
		// log the message
		log.Print(logMessage)
	}

	return
}
