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
	"sync"
	"time"
)

type debugInfo struct {
	last time.Time

	num_requests_received, num_requests_served, num_bytes_served int64

	sync.RWMutex
}

func (this *debugInfo) everything() (last_served_time time.Time, num_received, num_served, bytes_served int64) {
	this.RLock()
	last_served_time = this.last
	num_received = this.num_requests_received
	num_served = this.num_requests_served
	bytes_served = this.num_bytes_served
	this.RUnlock()
	return
}

func (this *debugInfo) requestServed(bytes_served int64) {
	this.Lock()
	this.num_requests_served++
	this.num_bytes_served += bytes_served
	this.last = time.Now()
	this.Unlock()
}
