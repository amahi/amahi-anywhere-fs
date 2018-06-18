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
	logging "log"
	"os"
)

const LOGFILE = "/var/log/amahi-anywhere.log"

var currentDebugLevel = 3

var logger *logging.Logger

func initializeLogging() {
	logFile, err := os.OpenFile(LOGFILE, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("WARNING: failed to open ", LOGFILE, " defaulting to standard output")
		logFile = os.Stdout
	}

	logger = logging.New(logFile, "", logging.LstdFlags)
}

func log(f string, args ...interface{}) {
	logger.Printf(f, args...)
}

func debugLevel(level int) {
	currentDebugLevel = level
}

func debug(level int, f string, args ...interface{}) {
	if PRODUCTION {
		return
	}
	if level <= currentDebugLevel {
		log(f, args...)
	}
}
