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
)

const LocalServerPort = "4563"

func (service *MercuryFsService) startLocalServer() {

	addr, err := net.ResolveTCPAddr("tcp", ":"+LocalServerPort)
	if err != nil {
		logError("Could not resolve local address")
		debug(2, "Error resolving local address: %s", err.Error())
		return
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		logError("Local server could not be started")
		debug(2, "Error on ListenTCP: %s", err.Error())
		return
	}
	defer listener.Close()

	for {
		logInfo("Starting local file server")
		debug(2, "Starting local file server at: %s", LocalServerPort)
		err = service.server.Serve(listener)
		if err != nil {
			logWarn("An error occurred in the local file server")
			debug(2, "local file server: %s", err.Error())
		}
	}
}
