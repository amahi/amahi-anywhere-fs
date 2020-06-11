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

func (apps *HdaApps) list() error {
	dbconn, err := sql.Open("mysql", MYSQL_CREDENTIALS)
	if err != nil {
		//log(err.Error())
		logging.Error(err.Error())
		return err
	}
	defer dbconn.Close()
	q := SQL_SELECT_APPS
	rows, err := dbconn.Query(q)
	if err != nil {
		//log(err.Error())
		logging.Error(err.Error())
		return err
	}
	newApps := make([]*HdaApp, 0)
	for rows.Next() {
		app := new(HdaApp)
		rows.Scan(&app.Vhost, &app.Name, &app.Logo)
		newApps = append(newApps, app)
	}

	apps.Lock()
	apps.Apps = newApps
	apps.Unlock()

	return nil
}

func (apps *HdaApps) get(vhost string) *HdaApp {
	for i := range apps.Apps {
		if apps.Apps[i].Vhost == vhost {
			return apps.Apps[i]
		}
	}
	return nil
}

func (apps *HdaApps) toJson() string {
	if len(apps.Apps) < 1 {
		return "[]"
	}

	ss := make([]string, 0)
	apps.RLock()

	// start by showing the dashboard first
	ss = append(ss, `{ "name": "Dashboard", "vhost": "hda", "logo": "https://wiki.amahi.org/images/8/8a/Dashboard-logo.png" }`)

	for i := range apps.Apps {
		temp := "{"
		// NB: 'name' and 'mtime' are used because of API spec
		if apps.Apps[i].Name != "" {
			temp += fmt.Sprintf(`"name": "%s", `, apps.Apps[i].Name)
		} else {
			temp += fmt.Sprintf(`"name": "%s", `, apps.Apps[i].Vhost)
		}
		temp += fmt.Sprintf(`"vhost": "%s", `, apps.Apps[i].Vhost)
		temp += fmt.Sprintf(`"logo": "%s"`, apps.Apps[i].Logo)
		temp += "}"
		ss = append(ss, temp)
	}

	apps.RUnlock()
	result := "[\n  "
	result += strings.Join(ss, ",\n  ")
	result += "\n]"
	return result
}
