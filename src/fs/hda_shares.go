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
	id         int
	name       string
	updatedAt  time.Time
	path       string
	tags       string
	isWritable bool
}

type HdaShares struct {
	Shares      []*HdaShare
	LastChecked time.Time
	sync.RWMutex
	root_dir    string
}

func NewHdaShares(root_dir string) (*HdaShares, error) {
	result := new(HdaShares)
	result.root_dir = root_dir
	err := result.update_shares()

	return result, err
}

func (this *HdaShares) update_shares() error {
	if this.root_dir == "" {
		return this.update_sql_shares()
	} else {
		return this.update_dir_shares()
	}
}

func (this *HdaShares) update_sql_shares() error {
	dbconn, err := sql.Open("mysql", MYSQL_CREDENTIALS)
	if err != nil {
		log(err.Error())
		return err
	}
	defer dbconn.Close()
	q := SQL_SELECT_SHARES
	debug(5, "share query: %s\n", q)
	rows, err := dbconn.Query(q)
	if err != nil {
		log(err.Error())
		return err
	}
	newShares := make([]*HdaShare, 0)
	for rows.Next() {
		share := new(HdaShare)
		rows.Scan(&share.name, &share.updatedAt, &share.path, &share.tags)
		debug(5, "share found: %s\n", share.name)
		newShares = append(newShares, share)
	}

	this.Lock()
	this.LastChecked = time.Now()
	this.Shares = newShares
	this.Unlock()

	return nil
}

func (this *HdaShares) update_dir_shares() (nil error) {

	dir, err := os.Open(this.root_dir)
	if err != nil {
		log(err.Error())
		return err
	}
	defer dir.Close()

	stat, _ := dir.Stat()
	if !stat.IsDir() {
		return errors.New("root_dir is not a directory")
	}

	fis, err := dir.Readdir(0)
	if err != nil {
		return err
	}

	newShares := make([]*HdaShare, 0)
	for i := range fis {
		if fis[i].IsDir() && strings.Index(fis[i].Name(), ".") != 0 {
			share := new(HdaShare)
			share.id = i
			share.name = fis[i].Name()
			share.updatedAt = fis[i].ModTime()
			share.tags = fis[i].Name()
			prefix, _ := filepath.Abs(this.root_dir)
			share.path = prefix + "/" + fis[i].Name()
			share.isWritable = true
			newShares = append(newShares, share)
		}
	}

	this.Lock()
	this.LastChecked = time.Now()
	this.Shares = newShares
	this.Unlock()

	return
}

func (this *HdaShares) Get(shareName string) *HdaShare {
	for i := range this.Shares {
		if this.Shares[i].name == shareName {
			return this.Shares[i]
		}
	}
	return nil
}

func SharesJson(shares []*HdaShare) string {
	if len(shares) < 1 {
		return "[]"
	}

	ss := []string{}

	for i := range shares {
		temp := "{"
		// NB: 'name' and 'mtime' are used because of API spec
		temp += fmt.Sprintf(`"name": "%s", `, shares[i].name)
		temp += fmt.Sprintf(`"mtime": "%s", `, shares[i].updatedAt.Format(http.TimeFormat))
		temp += fmt.Sprintf(`"tags": [%s],`, strings.Join(shares[i].tags_list(), ", "))
		temp += fmt.Sprintf(`"is_writable": %t`, shares[i].isWritable)
		temp += "}"
		ss = append(ss, temp)
	}

	result := "[\n  "
	result += strings.Join(ss, ",\n  ")
	result += "\n]"
	return result
}

// external interface to the path of a share
func (s *HdaShare) Path() string {
	return s.path
}

// return a list of tags, cleaned up
func (s *HdaShare) tags_list() []string {
	re := regexp.MustCompile(`(\s*,+\s*)+`)
	ta := re.Split(s.tags, -1)
	r := []string{};
	for _, tag := range ta {
		if tag != "" {
			r = append(r, fmt.Sprintf(`"%s"`, strings.TrimSpace(tag)))
		}
	}
	return r
}

// start a metadata pre-fill of the database in the background
func (this *HdaShares) start_metadata_prefill(library *metadata.Library) {
	// start it up after some time, to prevent overloads
	time.Sleep(15 * time.Second)
	for i := range this.Shares {
		path := this.Shares[i].path
		tags := strings.ToLower(this.Shares[i].tags)
		debug(5, `checking share "%s" (%s)  with tags: %s\n`, this.Shares[i].name, path, tags)
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
