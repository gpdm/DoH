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

import "github.com/sirupsen/logrus"

const (
	// LogEmerg is a Syslog-type priority for Emergency messages
	LogEmerg uint = iota

	// LogAlert is a Syslog-type priority for Alert messages
	LogAlert

	// LogCrit is a Syslog-type priority for Critical messages
	LogCrit

	// LogErr is a Syslog-type priority for Error messages
	LogErr

	// LogWarn is a Syslog-type priority for Warning messages
	LogWarn

	// LogNotice is a Syslog-type priority for Notice messages
	LogNotice

	// LogInform is a Syslog-type priority for Informational messages
	LogInform

	// LogDebug is a Syslog-type priority for Debug messages
	LogDebug
)

// LogLevels is a map of logrus <-> syslog log levels.
var LogLevels = map[uint]logrus.Level{
	LogEmerg:  logrus.PanicLevel,
	LogAlert:  logrus.FatalLevel,
	LogCrit:   logrus.FatalLevel,
	LogErr:    logrus.ErrorLevel,
	LogWarn:   logrus.WarnLevel,
	LogNotice: logrus.InfoLevel,
	LogInform: logrus.InfoLevel,
	LogDebug:  logrus.DebugLevel,
}
