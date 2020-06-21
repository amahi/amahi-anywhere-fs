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
	"crypto/tls"
	"github.com/fsnotify/fsnotify"

	// this is required for the side effect that it will register sha384/512 algorithms.
	// should not be needed in the future https://codereview.appspot.com/87670045/
	_ "crypto/sha512"
	"errors"
	"flag"
	"fmt"
	"github.com/amahi/go-metadata"
	"golang.org/x/net/http2"
	"hda_api_key"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// DANGER DANGER DANGER
// compile-time only options in case we need to disable checking the certs or https
const DisableCertChecking = false
const DisableHttps = false

const VERSION = "2.2"

var noDelete = false
var noUpload = false

// watcher for watching changes in files
var watcher *fsnotify.Watcher

// profiling info
// func init() { go func() { http.ListenAndServe(":4242", nil) }() }

func main() {

	defer panicHandler()

	setup()

	var dbg = 1
	var http2Debug = false
	var apiKeyFlag = ""
	var rootDir = ""
	var localAddr = ""
	var relayHost = PFE_HOST
	var relayPort = PFE_PORT
	var isDemo = false

	// Parse the program inputs
	if !PRODUCTION {
		flag.IntVar(&dbg, "d", 1, "print debug information, 1 = nothing printed and 5 = print everything")
		flag.BoolVar(&http2Debug, "h2", false, "HTTP2 debug")
		flag.StringVar(&apiKeyFlag, "k", "", "session token used by pfe")
		flag.StringVar(&rootDir, "r", "", "Use the directories in this directory as shares, instead of the registered HDA shares")
		flag.StringVar(&localAddr, "l", "", "Use this as the local address of the HDA, or look it up")
		flag.StringVar(&relayHost, "pfe", PFE_HOST, "address of the pfe")
		flag.StringVar(&relayPort, "pfe-port", PFE_PORT, "port the pfe is using")
		flag.BoolVar(&noDelete, "nd", false, "ignore delete requests silently")
		flag.BoolVar(&noUpload, "nu", false, "ignore upload requests silently")
		flag.BoolVar(&noBuffer, "nb", false, "ignore buffer logging silently")
	}
	flag.Parse()

	if rootDir != "" {
		isDemo = true
	}

	apiKey := ""
	if PRODUCTION || (!PRODUCTION && (apiKeyFlag == "")) {
		// no command line override - get it from the db
		key, err := hda_api_key.HDA_API_key(MYSQL_CREDENTIALS)
		if err != nil {
			cleanQuit(2, "Amahi API key was not found")
		}
		apiKey = key
	} else {
		apiKey = apiKeyFlag
	}

	if dbg < 1 || dbg > 5 {
		flag.PrintDefaults()
		return
	}
	debugLevel(dbg)

	if noDelete {
		fmt.Printf("NOTICE: running without deleting content!\n")
	}
	if noUpload {
		fmt.Printf("NOTICE: running without uploading content!\n")
	}

	initializeLogging(LOGFILE, splitFile, noBuffer)

	metadata, err := metadata.Init(100000, METADATA_FILE, TMDB_API_KEY, TVRAGE_API_KEY, TVDB_API_KEY)
	if err != nil {
		fmt.Printf("Error initializing metadata library\n")
		os.Remove(PID_FILE)
		os.Exit(1)
	}

	service, err := NewMercuryFSService(rootDir, localAddr, isDemo)
	if err != nil {
		fmt.Printf("Error making service (%s, %s): %s\n", rootDir, localAddr, err.Error())
		os.Remove(PID_FILE)
		os.Exit(1)
	}
	// start ONE delayed, background metadata prefill of the cache
	service.metadata = metadata

	go service.Shares.startMetadataPrefill(metadata)

	watcher, _ = fsnotify.NewWatcher()
	defer watcher.Close()

	go service.Shares.createThumbnailCache()

	//log("Amahi Anywhere service v%s", VERSION)
	logging.Info("Amahi Anywhere service v%s", VERSION)
	debug(4, "using api-key %s", apiKey)

	if http2Debug {
		http2.VerboseLogs = true
	}

	runtime.GOMAXPROCS(1000)
	go service.startLocalServer()

	// Continually connect to the proxy and listen for requests
	// Reconnect if there is an error
	for {
		conn, err := contactPfe(relayHost, relayPort, apiKey, service)
		if err != nil {
			//log("Error contacting the proxy.")
			logging.Error("Error contacting the proxy.")
			debug(2, "Error contacting the proxy: %s", err)
		} else {
			err = service.StartServing(conn)
			if err != nil {
				//log("Error serving requests")
				logging.Error("Error serving requests")
				debug(2, "Error in StartServing: %s", err)
			}
		}
		// reconnect fairly quickly, with some randomness
		sleepTime := time.Duration(2000 + rand.Intn(2000))
		time.Sleep(sleepTime * time.Millisecond)
	}
	os.Remove(PID_FILE)
}

// connect to the proxy and send a POST request with the api-key
func contactPfe(relayHost, relayPort, apiKey string, service *MercuryFsService) (net.Conn, error) {

	relayLocation := relayHost + ":" + relayPort
	//log("Contacting Relay at: " + relayLocation)
	logging.Info("Contacting Relay at: " + relayLocation)
	addr, err := net.ResolveTCPAddr("tcp", relayLocation)
	if err != nil {
		debug(2, "Error with ResolveTCPAddr: %s", err)
		return nil, err
	}

	tcpConn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		debug(2, "Error with initial DialTCP: %s", err)
		return nil, err
	}

	tcpConn.SetKeepAlive(true)
	tcpConn.SetLinger(0)
	service.info.relay_addr = relayLocation

	service.TLSConfig = &tls.Config{ServerName: relayHost}

	if DisableCertChecking {
		warning := "WARNING WARNING WARNING: running without checking TLS certs!!"
		logging.Warning(warning)
		logging.Warning(warning)
		logging.Warning(warning)
		fmt.Println(warning)
		fmt.Println(warning)
		fmt.Println(warning)
		service.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// Send the api-key
	buf := strings.NewReader(service.info.to_json())
	request, err := http.NewRequest("PUT", "https://"+relayLocation+"/fs", buf)
	if err != nil {
		debug(2, "Error creating NewRequest:", err)
		return nil, err
	}

	request.Header.Add("Api-Key", apiKey)
	request.Header.Add("Authorization", fmt.Sprintf("Token %s", SECRET_TOKEN))
	rawRequest, _ := httputil.DumpRequest(request, true)
	debug(5, "%s", rawRequest)

	var client *httputil.ClientConn

	if DisableHttps {
		warning := "WARNING WARNING: running without TLS!!"
		logging.Warning(warning)
		fmt.Println(warning)
		conn := tcpConn
		client = httputil.NewClientConn(conn, nil)
	} else {
		conn := tls.Client(tcpConn, service.TLSConfig)
		client = httputil.NewClientConn(conn, nil)
	}

	response, err := client.Do(request)
	if err != nil {
		debug(2, "Error writing to connection with Do: %s", err)
		return nil, err
	}

	if response.StatusCode != 200 {
		msg := fmt.Sprintf("Got an error response: %s", response.Status)
		//log(msg)
		logging.Error("Got an error response: %s", response.Status)
		return nil, errors.New(msg)
	}

	//log("Connected to the proxy")
	logging.Info("Connected to the proxy")
	netCon, _ := client.Hijack()

	return netCon, nil
}

// Clean up and quit
func cleanQuit(exitCode int, message string) {
	fmt.Println("FATAL:", message)
	os.Exit(exitCode)
}

func panicHandler() {
	if v := recover(); v != nil {
		fmt.Println("PANIC:", v)
	}
	os.Remove(PID_FILE)
}

func setup() error {

	checkPidFile()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			//log("Exiting with %v", sig)
			logging.Info("Exiting with %v", sig)
			os.Remove(PID_FILE)
			os.Exit(1)
		}
	}()

	return ioutil.WriteFile(PID_FILE, []byte(strconv.Itoa(os.Getpid())), 0666)
}

func checkPidFile() {
	if !exists(PID_FILE) {
		return
	}

	stale := false

	f, err := os.Open(PID_FILE)
	if err == nil {
		pid := make([]byte, 25)
		c, err := f.Read(pid)
		if err == nil {
			v, _ := strconv.Atoi(string(pid[:c]))
			fmt.Printf("PID: %#v\n", v)
			if !exists(fmt.Sprintf("/proc/%s/stat", string(pid[:c]))) {
				// the process does not exist. pid file is stale
				// note: this works on systems with /proc/
				stale = true
				os.Remove(PID_FILE)
			}
		}
	}
	if stale {
		fmt.Printf("PID file exists, but it's stale. Continuing.\n")
	} else {
		fmt.Printf("PID file exists and process is running. Exiting.\n")
		os.Exit(1)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
