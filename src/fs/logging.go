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
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const LOGFILE = "/var/log/amahi-anywhere.log"

// split type
const (
	splitNone = iota
	splitFile
	splitDir
)

type fileHandler struct {
	File   *os.File
	buffer []string
}

type LogMsg struct {
	level   Level
	message string
}

type Logging struct {
	filePath     string
	splitType    int
	fileHandlers map[string]*fileHandler
	fileName     string
	noBuffer     bool
	level        Level
	logChan      chan *LogMsg
}

// AccessLogRecord struct for holding access log data.
type AccessLogRecord struct {
	Origin        string        `json:"origin"`
	RequestTime   time.Time     `json:"request_time"`
	Request       string        `json:"request"`
	Status        int           `json:"status"`
	BodyBytesSent int           `json:"body_bytes_sent"`
	ElapsedTime   time.Duration `json:"elapsed_time"`
	HTTPReferrer  string        `json:"http_referrer"`
	HTTPUserAgent string        `json:"http_user_agent"`
	UserConnected int           `json:"user_connected"`
}

var currentDebugLevel = 3
var mutex sync.Mutex
var noBuffer = false
var logging *Logging

func initializeLogging(filePath string, splitType int, noBuffer bool) {
	logging = &Logging{
		filePath:     filePath,
		splitType:    splitType,
		fileName:     filepath.Base(filePath),
		noBuffer:     noBuffer,
		fileHandlers: make(map[string]*fileHandler),
		logChan:      make(chan *LogMsg, 1024),
	}
	err := logging.initFile()

	if err != nil {
		panic(err)
		return
	}
}

// initFile splits log file according to splitType
func (l *Logging) initFile() error {

	splitType := l.splitType
	logPath := filepath.Dir(l.filePath)

	err := createPath(logPath, 0644)
	if err != nil {
		return err
	}

	if splitType == splitNone {
		logFile := filepath.Join(logPath, l.fileName)
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		l.fileHandlers["default"] = &fileHandler{
			File:   file,
			buffer: nil,
		}
	} else if splitType == splitFile {
		levels := []Level{LevelDebug, LevelTrace, LevelInfo, LevelWarn, LevelError, LevelFatal, Access}
		for _, level := range levels {
			levelStr := strings.ToLower(level.String())
			logFile := filepath.Join(logPath, "."+levelStr+"."+l.fileName)
			file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			l.fileHandlers[level.String()] = &fileHandler{
				File:   file,
				buffer: nil,
			}

		}
	} else if splitType == splitDir {
		levels := []Level{LevelDebug, LevelTrace, LevelInfo, LevelWarn, LevelError, LevelFatal, Access}
		for _, level := range levels {
			levelStr := strings.ToLower(level.String())
			sonLogPath := filepath.Join(logPath, levelStr)
			err := createPath(sonLogPath, 0644)
			if err != nil {
				return err
			}

			logFile := filepath.Join(sonLogPath, "."+levelStr+"."+l.fileName)
			file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			l.fileHandlers[level.String()] = &fileHandler{
				File: file,
			}
		}
	}
	go l.backgroundLog()
	return nil
}

func (l *Logging) Debug(format string, a ...interface{}) {
	l.outPut(LevelDebug, format, a...)
}

func (l *Logging) Info(format string, a ...interface{}) {
	l.outPut(LevelInfo, format, a...)
}

func (l *Logging) Warning(format string, a ...interface{}) {
	l.outPut(LevelWarn, format, a...)
}

func (l *Logging) Error(format string, a ...interface{}) {
	l.outPut(LevelError, format, a...)
}

func (l *Logging) Fatal(format string, a ...interface{}) {
	l.outPut(LevelFatal, format, a...)
}

func debugLevel(level int) {
	currentDebugLevel = level
}

func debug(level int, f string, args ...interface{}) {
	if PRODUCTION {
		return
	}
	if level <= currentDebugLevel {
		logging.Debug(f, args)
	}
}

func (l *Logging) outPut(level Level, format string, a ...interface{}) {
	now := time.Now().Format("2006-01-02 15:04:05")
	funcName, fileName, lineNo := getInfo(3)

	format = fmt.Sprintf(format, a...)

	str := fmt.Sprintf("[%s] [%s] [%s:%s:%d]  %s \n", now, level.String(), fileName, funcName, lineNo, format)
	select {
	case l.logChan <- &LogMsg{level: level, message: str}:
	default:

	}
}

// output logs asynchronously
func (l *Logging) backgroundLog() {
	var fileIndexStr string
	go func(l *Logging) {
		// write a file every 3 seconds
		d := time.Second * 3
		ticker := time.NewTicker(d)
		defer ticker.Stop()
		for {
			<-ticker.C
			l.flush()
		}
	}(l)
	for {
		select {
		case logMsg := <-l.logChan:

			level := logMsg.level
			msg := logMsg.message
			if _, ok := l.fileHandlers[level.String()]; ok {

				fileIndexStr = level.String()
			} else {
				fileIndexStr = "default"
			}

			if l.noBuffer {
				_, err := l.fileHandlers[fileIndexStr].File.WriteString(msg)
				if err != nil {
					panic(err)
				}
				continue
			}
			l.fileHandlers[fileIndexStr].buffer = append(l.fileHandlers[fileIndexStr].buffer, msg)
			if len(l.fileHandlers[fileIndexStr].buffer) >= 2*1024 {
				l.flush()
			}
		default:
			// if channel is nil, then sleep
			time.Sleep(time.Millisecond * 200)
		}
	}

}

func (l *Logging) flush() {
	mutex.Lock()
	defer mutex.Unlock()

	for _, handler := range l.fileHandlers {
		buffer := handler.buffer
		handler.buffer = make([]string, 0)
		if len(buffer) == 0 {
			continue
		}
		_, err := handler.File.WriteString(strings.Join(buffer, ""))
		if err != nil {
			panic(err)
		}
	}
}

func getInfo(skip int) (funcName, fileName string, lineNo int) {
	pc, file, lineNo, ok := runtime.Caller(skip)
	if !ok {
		fmt.Printf("runtime.Caller() failed \n")
		return
	}

	funcName = runtime.FuncForPC(pc).Name()
	funcName = strings.Split(funcName, ".")[1]
	fileName = path.Base(file)
	return
}

// if the directory already exists, return nil
// else call MkdirALL
func createPath(path string, perm os.FileMode) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else {
		err := os.MkdirAll(path, perm)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Logging) AccessLog(r *AccessLogRecord) {
	timeFormatted := r.RequestTime.Format("02/Jan/2006 03:04:05")
	apacheFormatPattern := "%s - - [%s] \"%s %d %d\" %f %s %s - - - %d"
	msg := fmt.Sprintf(apacheFormatPattern, r.Origin, timeFormatted, r.Request, r.Status, r.BodyBytesSent,
		r.ElapsedTime.Seconds(), r.HTTPReferrer, r.HTTPUserAgent, r.UserConnected)
	select {
	case l.logChan <- &LogMsg{level: Access, message: msg}:
	default:

	}
}
