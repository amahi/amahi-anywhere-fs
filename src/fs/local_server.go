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
	"net"
	"github.com/amahi/go-metadata"
)

const LocalServerPort = "4563"

func startLocalServer(rootDir string, metadata *metadata.Library) {
	service, err := NewMercuryFSService(rootDir, ":"+LocalServerPort)
	if err != nil {
		log(err.Error())
		return
	}
	service.metadata = metadata

	addr, err := net.ResolveTCPAddr("tcp", ":"+LocalServerPort)
	if err != nil {
		log("Could not resolve local address")
		debug(2, "Error resolving local address: %s", err.Error())
		return
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log("Local server could not be started")
		debug(2, "Error on ListenTCP: %s", err.Error())
		return
	}
	defer listener.Close()

	for {
		log("Starting local file server")
		debug(2, "Starting local file server at: %s", LocalServerPort)
		err = service.server.Serve(listener)
		if err != nil {
			log("An error occurred in the local file server")
			debug(2, "local file server: %s", err.Error())
		}
	}
}
