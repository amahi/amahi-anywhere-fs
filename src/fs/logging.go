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

var current_debug_level = 3

var logger *logging.Logger

func initialize_logging() {
	log_file, err := os.OpenFile(LOGFILE, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("WARNING: failed to open ", LOGFILE, " defaulting to standard output")
		log_file = os.Stdout
	}

	logger = logging.New(log_file, "", logging.LstdFlags)
}

func log(f string, args ...interface{}) {
	logger.Printf(f, args...)
}

func debug_level(level int) {
	current_debug_level = level
}

func debug(level int, f string, args ...interface{}) {
	if PRODUCTION {
		return
	}
	if level <= current_debug_level {
		log(f, args...)
	}
}
