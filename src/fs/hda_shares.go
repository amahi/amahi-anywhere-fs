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
	"errors"
	"fmt"
	"github.com/amahi/go-metadata"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type HdaShare struct {
	name      string
	updatedAt time.Time
	path      string
	tags      string
	writable  bool
}

type HdaShares struct {
	Shares      []*HdaShare
	LastChecked time.Time
	sync.RWMutex
	rootDir string
}

func NewHdaShares(rootDir string) (*HdaShares, error) {
	result := new(HdaShares)
	result.rootDir = rootDir
	err := result.updateShares()

	return result, err
}

func (shares *HdaShares) updateShares() error {
	if shares.rootDir == "" {
		return shares.updateSqlShares()
	} else {
		return shares.updateDirShares()
	}
}

func (shares *HdaShares) updateSqlShares() error {
	dbconn, err := sql.Open("mysql", MYSQL_CREDENTIALS)
	if err != nil {
		//log(err.Error())
		logging.Error(err.Error())
		return err
	}
	defer dbconn.Close()
	q := SQL_SELECT_SHARES
	debug(5, "share query: %s\n", q)
	rows, err := dbconn.Query(q)
	if err != nil {
		//log(err.Error())
		logging.Error(err.Error())
		return err
	}
	newShares := make([]*HdaShare, 0)
	for rows.Next() {
		share := new(HdaShare)
		rows.Scan(&share.name, &share.updatedAt, &share.path, &share.tags)
		debug(5, "share found: %s\n", share.name)
		newShares = append(newShares, share)
	}

	shares.Lock()
	shares.LastChecked = time.Now()
	shares.Shares = newShares
	shares.Unlock()

	return nil
}

func (shares *HdaShares) updateDirShares() (nil error) {

	dir, err := os.Open(shares.rootDir)
	if err != nil {
		//log(err.Error())
		logging.Error(err.Error())
		return err
	}
	defer dir.Close()

	stat, _ := dir.Stat()
	if !stat.IsDir() {
		return errors.New("rootDir is not a directory")
	}

	fis, err := dir.Readdir(0)
	if err != nil {
		return err
	}

	newShares := make([]*HdaShare, 0)
	for i := range fis {
		if fis[i].IsDir() && strings.Index(fis[i].Name(), ".") != 0 {
			share := new(HdaShare)
			share.name = fis[i].Name()
			share.updatedAt = fis[i].ModTime()
			share.tags = fis[i].Name()
			prefix, _ := filepath.Abs(shares.rootDir)
			share.path = prefix + "/" + fis[i].Name()
			share.writable = true
			newShares = append(newShares, share)
		}
	}

	shares.Lock()
	shares.LastChecked = time.Now()
	shares.Shares = newShares
	shares.Unlock()

	return
}

func (shares *HdaShares) Get(shareName string) *HdaShare {
	for i := range shares.Shares {
		if shares.Shares[i].name == shareName {
			return shares.Shares[i]
		}
	}
	return nil
}

func SharesJson(shares []*HdaShare) string {
	if len(shares) < 1 {
		return "[]"
	}

	ss := make([]string, 0)

	for i := range shares {
		temp := "{"
		// NB: 'name' and 'mtime' are used because of API spec
		temp += fmt.Sprintf(`"name": "%s", `, shares[i].name)
		temp += fmt.Sprintf(`"mtime": "%s", `, shares[i].updatedAt.Format(http.TimeFormat))
		temp += fmt.Sprintf(`"tags": [%s],`, strings.Join(shares[i].tagsList(), ", "))
		temp += fmt.Sprintf(`"writable": %t`, shares[i].writable)
		temp += "}"
		ss = append(ss, temp)
	}

	result := "[\n  "
	result += strings.Join(ss, ",\n  ")
	result += "\n]"
	return result
}

// external interface to the path of a share
func (s *HdaShare) GetPath() string {
	return s.path
}

// return a list of tags, cleaned up
func (s *HdaShare) tagsList() []string {
	re := regexp.MustCompile(`(\s*,+\s*)+`)
	ta := re.Split(s.tags, -1)
	r := make([]string, 0)
	for _, tag := range ta {
		if tag != "" {
			r = append(r, fmt.Sprintf(`"%s"`, strings.TrimSpace(tag)))
		}
	}
	return r
}

// start a metadata pre-fill of the database in the background
func (shares *HdaShares) startMetadataPrefill(library *metadata.Library) {
	// start it up after some time, to prevent overloads
	time.Sleep(15 * time.Second)
	for i := range shares.Shares {
		path := shares.Shares[i].path
		tags := strings.ToLower(shares.Shares[i].tags)
		debug(5, `checking share "%s" (%s)  with tags: %s\n`, shares.Shares[i].name, path, tags)
		if path == "" || tags == "" {
			continue
		}
		if strings.Contains(tags, "movie") {
			library.Prefill(path, "movie", 0, true)
		} else if strings.Contains(tags, "tv") {
			library.Prefill(path, "tv", 0, true)
		}
	}
}

func (shares *HdaShares) createThumbnailCache() {
	time.Sleep(2 * time.Second)
	go func() {
		for {
			select {
			// watch for events
			case event := <-watcher.Events:
				op := event.Op.String()
				//log("FSNOTIFY EVENT: `%s`, NAME: `%s`", op, event.Name)
				logging.Info("FSNOTIFY EVENT: `%s`, NAME: `%s`", op, event.Name)
				switch {
				case op == "CREATE" || op == "WRITE":
					fillCache(event.Name)
				case op == "REMOVE" || op == "RENAME":
					removeCache(event.Name)
				}

				// watch for errors
			case err := <-watcher.Errors:
				fmt.Println("ERROR", err)
			}
		}
	}()

	//log("Starting caching")
	logging.Info("Starting caching")
	for i := range shares.Shares {
		// get path of the shares
		path := shares.Shares[i].path
		if path == "" {
			continue
		}
		fillCache(path)
	}
}
