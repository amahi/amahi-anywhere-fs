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

	numRequestsReceived, numRequestsServed, numBytesServed int64

	sync.RWMutex
}

func (d *debugInfo) everything() (lastServedTime time.Time, numReceived, numServed, bytesServed int64) {
	d.RLock()
	lastServedTime = d.last
	numReceived = d.numRequestsReceived
	numServed = d.numRequestsServed
	bytesServed = d.numBytesServed
	d.RUnlock()
	return
}

func (d *debugInfo) requestServed(bytesServed int64) {
	d.Lock()
	d.numRequestsServed++
	d.numBytesServed += bytesServed
	d.last = time.Now()
	d.Unlock()
}
