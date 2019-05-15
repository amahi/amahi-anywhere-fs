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

const LOGFILE = "/var/log/amahi-anywhere.log"

func initializeLogging() {
	logFile, err := os.OpenFile(LOGFILE, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("WARNING: failed to open ", LOGFILE, " defaulting to standard output")
		logFile = os.Stdout
	}

	Formatter := new(log2.TextFormatter)
	Formatter.TimestampFormat = "02-01-2006 15:04:05"
	Formatter.FullTimestamp = true
	//Formatter.DisableColors = true
	log2.SetFormatter(Formatter)
	log2.SetOutput(logFile)
}

func setLogLevel(level log2.Level) {
	log2.SetLevel(level)
}

func getLogLevel() log2.Level {
	return log2.GetLevel()
}

func log_trace(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args)
	log2.Trace(msg)
}

func log_debug(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args)
	log2.Debug(msg)
}

func log_info(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args)
	log2.Info(msg)
}

func log_warn(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args)
	log2.Warn(msg)
}

func log_error(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args)
	log2.Error(msg)
}

func log_fatal(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args)
	log2.Fatal(msg)
}

func log_panic(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args)
	log2.Panic(msg)
}

//
//var logger *logging.Logger
//
//func initializeLogging() {
//	logFile, err := os.OpenFile(LOGFILE, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
//	if err != nil {
//		fmt.Println("WARNING: failed to open ", LOGFILE, " defaulting to standard output")
//		logFile = os.Stdout
//	}
//
//	Formatter := new(log2.TextFormatter)
//	Formatter.TimestampFormat = "02-01-2006 15:04:05"
//	Formatter.FullTimestamp = true
//	//Formatter.DisableColors = true
//	log2.SetFormatter(Formatter)
//	log2.SetOutput(logFile)
//
//	logger = logging.New(logFile, "", logging.LstdFlags)
//}
//
//func log(f string, args ...interface{}) {
//	logger.Printf(f, args...)
//}
//
//func debugLevel(level int) {
//	currentDebugLevel = level
//}
//
//func debug(level int, f string, args ...interface{}) {
//	if PRODUCTION {
//		return
//	}
//	if level <= currentDebugLevel {
//		log(f, args...)
//	}
//}
