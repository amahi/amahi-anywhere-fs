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
	"os"
	"testing"
)

func TestDirToJson(t *testing.T) {
	file, err := os.Open(".")
	if err != nil {
		t.Error(err.Error())
		return
	}
	defer file.Close()

	testData, err := dirToJSON(file)
	if err != nil {
		t.Error(err.Error())
		return
	} else if len(testData) <= 0 {
		t.Error("Empty data returned")
	}
	file.Close()
	file, err = os.Open(".")
	defer file.Close()

	_, err = os.Create(".test")
	if err != nil {
		t.Fatalf("Create failed: %s", err.Error())
	}
	defer os.Remove(".test")

	testData2, err := dirToJSON(file)
	if err != nil {
		t.Fatalf("Second dirToJSON failed: %s", err.Error())
	}

	if len(testData) != len(testData2) {
		t.Errorf("testData length (%d) does not equal testData2 length (%d)", len(testData), len(testData2))
	}
}

func TestGetContentType(t *testing.T) {
	testName := "test.pdf"

	test := getContentType(testName)

	if test != "application/pdf" {
		t.Errorf("Wrong output for %s: %s", testName, test)
		return
	}

	test = getContentType("")
	if test != "application/octet-stream" {
		t.Errorf("Wrong output for %s: %s", testName, test)
		return
	}
}
