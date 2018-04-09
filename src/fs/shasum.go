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
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
)

// compute the sha1-encoded value for the string
func sha1string(value string) string {
	sum := sha1.New()
	io.WriteString(sum, value)
	str := fmt.Sprintf("%x", sum.Sum(nil))
	return str
}

// compute the sha1-encoded value for the byte array
func sha1bytes(value []byte) string {
	sum := sha1.New()
	io.WriteString(sum, bytes.NewBuffer(value).String())
	str := fmt.Sprintf("%x", sum.Sum(nil))
	return str
}
