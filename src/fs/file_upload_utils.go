/*
 * Copyright (c) 2013-2019 Amahi
 *
 * This file is part of Amahi.
 *
 * Amahi is free software released under the GNU GPL v3 license.
 * See the LICENSE file accompanying this distribution.
 */

package main

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

const (
	FILE_NOT_EXISTS = iota
	FILE_EXISTS
	FILE_SAME_MD5
)

// rename the given file path
func renameFile(p string) string {
	//get the file suffix
	ext := path.Ext(p)
	//get the filename without suffix
	baseName := strings.Replace(p, ext, "", 1)

	timeStamp := time.Now().Format("20060102-1504")

	return baseName + "-" + timeStamp + ext
}

func checkFileExists(filename string, f io.Reader) int {
	_, err := os.Stat(filename)
	//file not exists
	if err != nil {
		return FILE_NOT_EXISTS
	}
	localFile, ferr := os.Open(filename)
	if ferr != nil {
		return FILE_NOT_EXISTS
	}

	//file exists, comparing the md5
	if calMD5(localFile) == calMD5(f) {
		return FILE_SAME_MD5
	}

	//file exists
	return FILE_EXISTS
}

//check if the file path is valid
func validFilename(f string) bool {
	// Check if file already exists
	if _, err := os.Stat(f); err == nil {
		return true
	}

	// Attempt to create it
	var d []byte
	if err := ioutil.WriteFile(f, d, 0644); err == nil {
		os.Remove(f) // And delete it
		return true
	}

	return false
}

//calculate the md5
func calMD5(input io.Reader) string {
	h := md5.New()
	io.Copy(h, input)
	return hex.EncodeToString(h.Sum(nil))
}
