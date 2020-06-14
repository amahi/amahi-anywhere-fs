package main

import (
	"testing"
	"time"
)

func TestLoggingSplitNone(t *testing.T) {
	initializeLogging("./temp/log/test.log",  splitNone, false)
	logging.Error("error")
	logging.Warning("warn")
	logging.Debug("debug")
	logging.Info("info")
	logging.Fatal("fatal")
	time.Sleep(5*time.Second)

}

func TestLoggingSplitFile(t *testing.T) {
	initializeLogging("./temp/log/test2.log",  splitFile, false)
	logging.Error("error")
	logging.Warning("warn")
	logging.Debug("debug")
	logging.Info("info")
	logging.Fatal("fatal")
	time.Sleep(5*time.Second)
}

func TestLoggingSplitDir(t *testing.T) {
	initializeLogging("./temp/log/test3.log",  splitDir, false)
	logging.Error("error")
	logging.Warning("warn")
	logging.Debug("debug")
	logging.Info("info")
	logging.Fatal("fatal")
	time.Sleep(5*time.Second)
}

