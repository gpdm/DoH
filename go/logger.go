/*
 * DoH Service - Logging Functions
 *
 * This is the logging collection, which provides some abstracts for log handling
 *
 * Contact: dev@phunsites.net
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
