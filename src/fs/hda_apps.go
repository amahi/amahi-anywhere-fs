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
	"database/sql"
	"fmt"
	"strings"
	"sync"
)

type HdaApp struct {
	Name  string
	Vhost string
	Logo  string
}

type HdaApps struct {
	Apps []*HdaApp
	sync.RWMutex
}

func newHdaApps() (*HdaApps, error) {
	result := new(HdaApps)
	err := result.list()

	return result, err
}

func (this *HdaApps) list() error {
	dbconn, err := sql.Open("mysql", MYSQL_CREDENTIALS)
	if err != nil {
		log(err.Error())
		return err
	}
	defer dbconn.Close()
	q := SQL_SELECT_APPS
	rows, err := dbconn.Query(q)
	if err != nil {
		log(err.Error())
		return err
	}
	newApps := make([]*HdaApp, 0)
	for rows.Next() {
		app := new(HdaApp)
		rows.Scan(&app.Vhost, &app.Name, &app.Logo)
		newApps = append(newApps, app)
	}

	this.Lock()
	this.Apps = newApps
	this.Unlock()

	return nil
}

func (this *HdaApps) get(vhost string) *HdaApp {
	for i := range this.Apps {
		if this.Apps[i].Vhost == vhost {
			return this.Apps[i]
		}
	}
	return nil
}

func (this *HdaApps) to_json() string {
	if len(this.Apps) < 1 {
		return "[]"
	}

	ss := []string{}
	this.RLock()

	// start by showing the dashboard first
	ss = append(ss, `{ "name": "Dashboard", "vhost": "hda", "logo": "https://wiki.amahi.org/images/8/8a/Dashboard-logo.png" }`)

	for i := range this.Apps {
		temp := "{"
		// NB: 'name' and 'mtime' are used because of API spec
		if this.Apps[i].Name != "" {
			temp += fmt.Sprintf(`"name": "%s", `, this.Apps[i].Name)
		} else {
			temp += fmt.Sprintf(`"name": "%s", `, this.Apps[i].Vhost)
		}
		temp += fmt.Sprintf(`"vhost": "%s", `, this.Apps[i].Vhost)
		temp += fmt.Sprintf(`"logo": "%s"`, this.Apps[i].Logo)
		temp += "}"
		ss = append(ss, temp)
	}

	this.RUnlock()
	result := "[\n  "
	result += strings.Join(ss, ",\n  ")
	result += "\n]"
	return result
}
