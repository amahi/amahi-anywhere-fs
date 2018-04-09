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
	"runtime"
)

type HdaInfo struct {
	version, local_addr, relay_addr string
}

func (this *HdaInfo) to_json() string {
	return fmt.Sprintf(`{"version": "%s", "local_addr": "%s", "relay_addr": "%s", "arch": "%s-%s-%d"}`, this.version, this.local_addr, this.relay_addr, runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
}
