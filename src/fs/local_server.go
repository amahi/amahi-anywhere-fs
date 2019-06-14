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
		//log("Could not resolve local address")
		//log2.Fatal("Could not resolve local address")
		log_fatal("Could not resolve local address")

		//debug(2, "Error resolving local address: %s", err.Error())
		//log2.Error(fmt.Sprintf("Error resolving local address: %s", err.Error()))
		log_error("Error resolving local address: %s", err.Error())
		return
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		//log("Local server could not be started")
		//log2.Fatal("Local server could not be started")
		log_fatal("Local server could not be started")

		//debug(2, "Error on ListenTCP: %s", err.Error())
		//log2.Debug(fmt.Sprintf("Error on ListenTCP: %s", err.Error()))
		log_error("Error on ListenTCP: %s", err.Error())
		return
	}
	defer listener.Close()

	for {
		//log("Starting local file server")
		//log2.Info("Starting local file server")
		log_info("Starting local file server")

		//debug(2, "Starting local file server at: %s", LocalServerPort)
		//log2.Debug(fmt.Sprintf("Starting local file server at: %s", LocalServerPort))
		log_info("Starting local file server at: %s", LocalServerPort)
		err = service.server.Serve(listener)
		if err != nil {
			//log("An error occurred in the local file server")
			//log2.Error("An error occurred in the local file server")
			log_warn("An error occurred in the local file server")
			//debug(2, "local file server: %s", err.Error())
			//log2.Debug(fmt.Sprintf(2, "local file server: %s", err.Error()))
			log_error("local file server: %s", err.Error())
		}
	}
}
