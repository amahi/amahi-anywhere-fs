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
	// this is required for the side effect that it will register sha384/512 algorithms.
	// should not be needed in the future https://codereview.appspot.com/87670045/
	_ "crypto/sha512"
	"errors"
	"flag"
	"fmt"
	"github.com/amahi/go-metadata"
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
	"golang.org/x/net/http2"
)

// DANGER DANGER DANGER
// compile-time only options in case we need to disable checking the certs or https
const DISABLE_CERT_CHECKING = false
const DISABLE_HTTPS = false

const VERSION = "1.70"

var no_delete = false
var no_upload = false

// profiling info
// func init() { go func() { http.ListenAndServe(":4242", nil) }() }

func main() {

	defer panic_handler()

	setup()

	var dbg = 1
	var http2_debug = false
	var api_key_flag = ""
	var root_dir = ""
	var local_addr = ""
	var relay_host = PFE_HOST
	var relay_port = PFE_PORT

	// Parse the program inputs
	if !PRODUCTION {
		flag.IntVar(&dbg, "d", 1, "print debug information, 1 = nothing printed and 5 = print everything")
		flag.BoolVar(&http2_debug, "h", false, "HTTP2 debug")
		flag.StringVar(&api_key_flag, "k", "", "session token used by pfe")
		flag.StringVar(&root_dir, "r", "", "Use the directories in this directory as shares, instead of the registered HDA shares")
		flag.StringVar(&local_addr, "l", "", "Use this as the local address of the HDA, or look it up")
		flag.StringVar(&relay_host, "pfe", PFE_HOST, "address of the pfe")
		flag.StringVar(&relay_port, "pfe-port", PFE_PORT, "port the pfe is using")
		flag.BoolVar(&no_delete, "nd", false, "ignore delete requests silently")
		flag.BoolVar(&no_upload, "nu", false, "ignore upload requests silently")
	}
	flag.Parse()

	api_key := ""
	if PRODUCTION || (!PRODUCTION && (api_key_flag == "")) {
		// no command line override - get it from the db
		key, err := hda_api_key.HDA_API_key(MYSQL_CREDENTIALS)
		if err != nil {
			cleanQuit(2, "Amahi API key was not found")
		}
		api_key = key
	} else {
		api_key = api_key_flag
	}

	if dbg < 1 || dbg > 5 {
		flag.PrintDefaults()
		return
	}
	debug_level(dbg)

	if (no_delete) { fmt.Printf("NOTICE: running without deleting content!\n") }
	if (no_upload) { fmt.Printf("NOTICE: running without uploading content!\n") }

	initialize_logging()

	metadata, err := metadata.Init(100000, METADATA_FILE, TMDB_API_KEY, TVRAGE_API_KEY, TVDB_API_KEY)
	if err != nil {
		fmt.Printf("Error initializing metadata library\n")
		os.Remove(PID_FILE)
		os.Exit(1)
	}

	service, err := NewMercuryFSService(root_dir, local_addr)
	if err != nil {
		fmt.Printf("Error making service (%s, %s): %s\n", root_dir, local_addr, err.Error())
		os.Remove(PID_FILE)
		os.Exit(1)
	}
	// start ONE delayed, background metadata prefill of the cache
	service.metadata = metadata

	go service.Shares.start_metadata_prefill(metadata)

	log("Amahi Anywhere service v%s", VERSION)

	debug(4, "using api-key %s", api_key)

	if http2_debug {
		http2.VerboseLogs = true
	}

	runtime.GOMAXPROCS(1000)
	go start_local_server(root_dir, metadata)

	// Continually connect to the proxy and listen for requests
	// Reconnect if there is an error
	for {
		conn, err := contact_pfe(relay_host, relay_port, api_key, service)
		if err != nil {
			log("Error contacting the proxy.")
			debug(2, "Error contacting the proxy: %s", err)
		} else {
			err = service.StartServing(conn)
			if err != nil {
				log("Error serving requests")
				debug(2, "Error in StartServing: %s", err)
			}
		}
		// reconnect fairly quickly, with some randomness
		sleep_time := time.Duration(2000 + rand.Intn(2000))
		time.Sleep(sleep_time * time.Millisecond)
	}
	os.Remove(PID_FILE)
}

// connect to the proxy and send a POST request with the api-key
func contact_pfe(relay_host, relay_port, api_key string, service *MercuryFsService) (net.Conn, error) {

	relay_location := relay_host + ":" + relay_port
	log("Contacting Relay at: " + relay_location)
	addr, err := net.ResolveTCPAddr("tcp", relay_location)
	if err != nil {
		debug(2, "Error with ResolveTCPAddr: %s", err)
		return nil, err
	}

	tcp_conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		debug(2, "Error with initial DialTCP: %s", err)
		return nil, err
	}

	tcp_conn.SetKeepAlive(true)
	tcp_conn.SetLinger(0)
	service.info.relay_addr = relay_location

	service.TLSConfig = &tls.Config{ ServerName: relay_host }

	if DISABLE_CERT_CHECKING {
		warning := "WARNING WARNING WARNING: running without checking TLS certs!!"
		log(warning)
		log(warning)
		log(warning)
		fmt.Println(warning)
		fmt.Println(warning)
		fmt.Println(warning)
		service.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// Send the api-key
	buf := strings.NewReader(service.info.to_json())
	request, err := http.NewRequest("PUT", "https://"+relay_location+"/fs", buf)
	if err != nil {
		debug(2, "Error creating NewRequest:", err)
		return nil, err
	}

	request.Header.Add("Api-Key", api_key)
	request.Header.Add("Authorization", fmt.Sprintf("Token %s", SECRET_TOKEN))
	raw_request, _ := httputil.DumpRequest(request, true)
	debug(5, "%s", raw_request)

	var client *httputil.ClientConn

	if DISABLE_HTTPS {
		warning := "WARNING WARNING: running without TLS!!"
		log(warning)
		fmt.Println(warning)
		conn := tcp_conn
		client = httputil.NewClientConn(conn, nil)
	} else {
		conn := tls.Client(tcp_conn, service.TLSConfig)
		client = httputil.NewClientConn(conn, nil)
	}

	response, err := client.Do(request)
	if err != nil {
		debug(2, "Error writing to connection with Do: %s", err)
		return nil, err
	}

	if response.StatusCode != 200 {
		msg := fmt.Sprintf("Got an error response: %s", response.Status)
		log(msg)
		return nil, errors.New(msg)
	}

	log("Connected to the proxy")

	net_con, _ := client.Hijack()

	return net_con, nil
}

// Clean up and quit
func cleanQuit(exitCode int, message string) {
	fmt.Println("FATAL:", message)
	os.Exit(exitCode)
}

func panic_handler() {
	if v := recover(); v != nil {
		fmt.Println("PANIC:", v)
	}
	os.Remove(PID_FILE)
}

func setup() error {

	check_pid_file()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log("Exiting with %v", sig)
			os.Remove(PID_FILE)
			os.Exit(1)
		}
	}()

	return ioutil.WriteFile(PID_FILE, []byte(strconv.Itoa(os.Getpid())), 0666)
}

func check_pid_file() {
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
