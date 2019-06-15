/*
 * Copyright (c) 2013-2018 Amahi
 *
 * This file is part of Amahi.
 *
 * Amahi is free software released under the GNU GPL v3 license.
 * See the LICENSE file accompanying this distribution.
 */

package main

import (
	"fmt"
	log2 "github.com/Sirupsen/logrus"
	"os"
)

//const LOGFILE = "/var/log/amahi-anywhere.log"
const LOGFILE = "temp_log.log"

func initializeLogging() {
	logFile, err := os.OpenFile(LOGFILE, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("WARNING: failed to open ", LOGFILE, " defaulting to standard output")
		logFile = os.Stdout
	}

	Formatter := new(log2.TextFormatter)
	Formatter.TimestampFormat = "02-01-2006 15:04:05"
	Formatter.FullTimestamp = true
	Formatter.DisableColors = true
	log2.SetFormatter(Formatter)
	log2.SetOutput(logFile)
}

func setLogLevel(level log2.Level) {
	log2.SetLevel(level)
}

func getLogLevel() log2.Level {
	return log2.GetLevel()
}

func logTrace(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args...)
	log2.Trace(msg)
}

func logDebug(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args...)
	log2.Debug(msg)
}

func logInfo(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args...)
	log2.Info(msg)
}

func logWarn(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args...)
	log2.Warn(msg)
}

func logError(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args...)
	log2.Error(msg)
}

func logFatal(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args...)
	log2.Fatal(msg)
}

func logPanic(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args...)
	log2.Panic(msg)
}

func logHttp(method, endpoint string, responseCode, responseSize int, ua string) {
	//having a separate method for logging will help easily modify the log statements if required
	logInfo("\"%s %s\" %d %d \"%s\"", method, endpoint, responseCode, responseSize, ua)
}
