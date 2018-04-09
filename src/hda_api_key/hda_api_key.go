/*
 * Copyright (c) 2013-2018 Amahi
 *
 * This file is part of Amahi.
 *
 * Amahi is free software released under the GNU GPL v3 license.
 * See the LICENSE file accompanying this distribution.
 */

package hda_api_key

import (
	"crypto/sha1"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
)

func HDA_API_key(credentials string) (string, error) {

	dbconn, err := sql.Open("mysql", credentials)
	if err != nil {
		return "", err
	}
	defer dbconn.Close()

	// query the database
	var api_key string
	q := "select value from settings where name='api-key'"
	rows, err := dbconn.Query(q)
	if err != nil {
		return "", err
	}

	// get the first api key found -- hopefully there is only one
	for rows.Next() {
		err = rows.Scan(&api_key)
		if err != nil {
			return "", err
		}
		// fmt.Println (api_key)
		// compute the sha1-encoded value for the API key
		sum := sha1.New()
		io.WriteString(sum, api_key)
		str := fmt.Sprintf("%x", sum.Sum(nil))
		// return not the API key, but the sha1-encoded value for it
		return str, nil
	}

	return "", err
}
