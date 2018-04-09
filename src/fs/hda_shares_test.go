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

func TestUpdateShares(t *testing.T) {
	// NewHdaShares calls update_shares
	test, err := NewHdaShares(".")
	if err != nil {
		t.Errorf(". test failed: %s", err.Error())
		return
	} else if len(test.Shares) != 0 {
		t.Errorf("Expected 0 shares but got %d shares", len(test.Shares))
	}

	err = os.MkdirAll("test/1", 0777)
	if err != nil {
		t.Errorf("Mkdir failed: %s", err.Error())
		return
	}
	defer os.RemoveAll("test")
	_, err = os.Create("test/.test")
	if err != nil {
		t.Fatalf("Creation of .test failed: %s", err.Error())
	}

	test, err = NewHdaShares("test")
	if err != nil {
		t.Errorf("test test failed: %s", err.Error())
		return
	} else if len(test.Shares) != 1 {
		t.Errorf("Expected 1 shares but got %d shares", len(test.Shares))
	}
}
