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
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"github.com/amahi/go-metadata"
	"github.com/gorilla/mux"
	"golang.org/x/net/http2"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// MercuryFsService defines the file server and directory server API
type MercuryFsService struct {
	Users  *HdaUsers
	Shares *HdaShares
	Apps   *HdaApps

	// TLS configuration
	TLSConfig *tls.Config

	// http server hooks
	server *http.Server

	info *HdaInfo

	metadata *metadata.Library

	debugInfo *debugInfo

	apiRouter *mux.Router
}

// NewMercuryFsService creates a new MercuryFsService, sets the FileDirectoryRoot
// and CurrentDirectory to rootDirectory and returns the pointer to the
// newly created MercuryFsService
func NewMercuryFSService(rootDir, localAddr string, isDemo bool) (service *MercuryFsService, err error) {
	service = new(MercuryFsService)

	service.Users = NewHdaUsers(isDemo)
	service.Shares, err = NewHdaShares(rootDir)
	if err != nil {
		logError("Error making HdaShares: %s", err.Error())
		return nil, err
	}
	service.debugInfo = new(debugInfo)

	// set up API mux
	apiRouter := mux.NewRouter()
	apiRouter.HandleFunc("/auth", service.authenticate).Methods("POST")
	apiRouter.HandleFunc("/logout", service.logout).Methods("POST")
	apiRouter.HandleFunc("/shares", service.serveShares).Methods("GET")
	apiRouter.HandleFunc("/files", use(service.serveFile, service.shareReadAccess, service.restrictCache)).Methods("GET")
	apiRouter.HandleFunc("/files", use(service.deleteFile, service.shareWriteAccess, service.restrictCache)).Methods("DELETE")
	apiRouter.HandleFunc("/files", use(service.uploadFile, service.shareWriteAccess, service.restrictCache)).Methods("POST")
	apiRouter.HandleFunc("/cache", use(service.serveCache, service.shareReadAccess)).Methods("GET")
	apiRouter.HandleFunc("/apps", service.appsList).Methods("GET")
	apiRouter.HandleFunc("/md", service.getMetadata).Methods("GET")
	apiRouter.HandleFunc("/hda_debug", service.hdaDebug).Methods("GET")
	apiRouter.HandleFunc("/logs", service.serveLogs).Methods("GET")

	apiRouter.MethodNotAllowedHandler = http.HandlerFunc(service.methodNotAllowedHandler)
	apiRouter.NotFoundHandler = http.HandlerFunc(service.notFoundHandler)

	service.apiRouter = apiRouter

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", http.HandlerFunc(service.topVhostFilter))

	service.server = &http.Server{TLSConfig: service.TLSConfig, Handler: serveMux}

	service.info = new(HdaInfo)
	service.info.version = VERSION
	if localAddr != "" {
		service.info.local_addr = localAddr
	} else {
		actualAddr, err := GetLocalAddr(rootDir)
		if err != nil {
			debug(2, "Error getting local address: %s", err.Error())
			return nil, err
		}
		service.info.local_addr = actualAddr + ":" + LocalServerPort
	}
	// This will be set when the HDA connects to the proxy
	service.info.relay_addr = ""

	debug(3, "Amahi FS Service started %s", SharesJson(service.Shares.Shares))
	debug(4, "HDA Info: %s", service.info.to_json())

	return service, err
}

// String returns FileDirectoryRoot and CurrentDirectory with a newline between them
func (service *MercuryFsService) String() string {
	// TODO: Possibly change this to present a more formatted string
	return SharesJson(service.Shares.Shares)
}

//type notFoundHandler struct {}
func (service *MercuryFsService) notFoundHandler(writer http.ResponseWriter, request *http.Request) {
	errMsg := "404 page not found"
	writer.WriteHeader(http.StatusNotFound)
	writer.Write([]byte(errMsg))
	logHttp(service, request, http.StatusNotFound, len(errMsg))
}

//type methodNotAllowedHandler struct{}
func (service *MercuryFsService) methodNotAllowedHandler(writer http.ResponseWriter, request *http.Request) {
	errMsg := "405 method not allowed"
	writer.WriteHeader(http.StatusMethodNotAllowed)
	writer.Write([]byte(errMsg))
	logHttp(service, request, http.StatusMethodNotAllowed, len(errMsg))
}

func (service *MercuryFsService) hdaDebug(writer http.ResponseWriter, request *http.Request) {
	// I am purposely not calling any of the update methods of debugInfo to actually provide valuable info
	result := "{\n"
	result += fmt.Sprintf("\"goroutines\": %d\n", runtime.NumGoroutine())
	relayAddr := service.info.relay_addr
	result += `"connected": `
	if relayAddr != "" {
		result += "true\n"
	} else {
		result += "false\n"
	}
	last, received, served, numBytes := service.debugInfo.everything()
	actualDate := ""
	if served != 0 {
		actualDate = last.Format(http.TimeFormat)
	}
	outstanding := received - served
	if outstanding < 0 {
		outstanding = 0
	}
	result += fmt.Sprintf("\"last_request\": \"%s\"\n", actualDate)
	result += fmt.Sprintf("\"received\": %d\n", received)
	result += fmt.Sprintf("\"served\": %d\n", served)
	result += fmt.Sprintf("\"outstanding\": %d\n", outstanding)
	result += fmt.Sprintf("\"bytes_served\": %d\n", numBytes)

	result += "}"
	writer.WriteHeader(200)
	size, _ := writer.Write([]byte(result))
	logHttp(service, request, 200, size)
}

func directory(fi os.FileInfo, js string, w http.ResponseWriter, request *http.Request) (status, size int64) {
	json := []byte(js)
	etag := `"` + sha1bytes(json) + `"`
	w.Header().Set("ETag", etag)
	inm := request.Header.Get("If-None-Match")
	if inm == etag {
		size = 0
		debug(4, "If-None-Match match found for %s", etag)
		w.WriteHeader(http.StatusNotModified)
		status = 304
	} else {
		debug(4, "If-None-Match (%s) match NOT found for Etag %s", inm, etag)
		size = int64(len(json))
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		w.Header().Set("Last-Modified", fi.ModTime().UTC().Format(http.TimeFormat))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "max-age=0, private, must-revalidate")
		w.WriteHeader(http.StatusOK)
		w.Write(json)
		status = 200
	}
	return status, size
}

// fullPathToFile creates the full path to the requested file and checks to make sure that
// there aren't any  '..' to prevent unauthorized access
func (service *MercuryFsService) fullPathToFile(shareName, relativePath string) (string, error) {
	share := service.Shares.Get(shareName)
	if share == nil {
		return "", errors.New(fmt.Sprintf("Share %s not found", shareName))
	} else if strings.Contains(relativePath, "../") {
		return "", errors.New(fmt.Sprintf("path %s contains ..", relativePath))
	}

	path := share.GetPath() + relativePath
	return path, nil
}

// serve requests with the ServeConn function over HTTP/2, in goroutines, until we get some error
func (service *MercuryFsService) StartServing(conn net.Conn) error {
	logInfo("Connection to the proxy established.")

	service.info.relay_addr = conn.RemoteAddr().String()
	serveConnOpts := &http2.ServeConnOpts{BaseConfig: service.server}
	server2 := new(http2.Server)

	// start serving over http2 on provided conn and block until connection is lost
	server2.ServeConn(conn, serveConnOpts)

	logWarn("Lost connection to the proxy.")
	service.info.relay_addr = ""
	return errors.New("connection is no longer readable")
}

func (service *MercuryFsService) serveFile(writer http.ResponseWriter, request *http.Request) {
	q := request.URL
	path := q.Query().Get("p")
	share := q.Query().Get("s")

	debug(2, "serve_file GET request")

	service.printRequest(request)

	fullPath, err := service.fullPathToFile(share, path)
	if err != nil {
		debug(2, "File not found: %s", err)
		http.NotFound(writer, request)
		service.debugInfo.requestServed(int64(0))
		logHttp(service, request, 404, 0)
		return
	}
	osFile, err := os.Open(fullPath)
	if err != nil {
		debug(2, "Error opening file: %s", err.Error())
		http.NotFound(writer, request)
		service.debugInfo.requestServed(int64(0))
		logHttp(service, request, 404, 0)
		return
	}
	defer osFile.Close()

	// This shouldn't return an error since we just opened the file
	fi, _ := osFile.Stat()

	// If the file is a directory, return the all the files within the directory...
	if fi.IsDir() || isSymlinkDir(fi, fullPath) {
		jsonDir, err := dirToJSON(osFile, fullPath, share, path)
		if err != nil {
			debug(2, "Error converting dir to JSON: %s", err.Error())
			http.NotFound(writer, request)
			service.debugInfo.requestServed(int64(0))
			logHttp(service, request, 404, 0)
			return
		}
		debug(5, "%s", jsonDir)
		status, size := directory(fi, jsonDir, writer, request)
		service.debugInfo.requestServed(size)
		logHttp(service, request, int(status), int(size))
		return
	}

	// we use for etag the sha1sum of the full path followed the mtime
	mtime := fi.ModTime().UTC().Format(http.TimeFormat)
	etag := `"` + sha1string(path+mtime) + `"`
	inm := request.Header.Get("If-None-Match")
	if inm == etag {
		debug(4, "If-None-Match match found for %s", etag)
		writer.WriteHeader(http.StatusNotModified)
		logHttp(service, request, 304, 0)
	} else {
		writer.Header().Set("Last-Modified", mtime)
		writer.Header().Set("ETag", etag)
		writer.Header().Set("Cache-Control", "max-age=0, private, must-revalidate")
		debug(4, "Etag sent: %s", etag)
		http.ServeContent(writer, request, fullPath, fi.ModTime(), osFile)
		service.debugInfo.requestServed(fi.Size())
		logHttp(service, request, 200, int(fi.Size()))
	}
	return
}

func (service *MercuryFsService) serveCache(writer http.ResponseWriter, request *http.Request) {
	q := request.URL
	path := q.Query().Get("p")
	share := q.Query().Get("s")

	debug(2, "serve_file GET request")

	service.printRequest(request)

	fullPath, err := service.fullPathToFile(share, path)

	parentDir := filepath.Dir(fullPath)
	filename := filepath.Base(fullPath)
	thumbnailDirPath := filepath.Join(parentDir, ".fscache/thumbnails")
	thumbnailPath := filepath.Join(thumbnailDirPath, filename)

	if err != nil {
		debug(2, "File not found: %s", err)
		http.NotFound(writer, request)
		service.debugInfo.requestServed(int64(0))
		logHttp(service, request, 404, 0)
		return
	}
	osFile, err := os.Open(thumbnailPath)
	if err != nil {
		debug(2, "Error opening cache file: %s", err.Error())
		http.NotFound(writer, request)
		service.debugInfo.requestServed(int64(0))
		logHttp(service, request, 404, 0)
		return
	}
	defer osFile.Close()

	// This shouldn't return an error since we just opened the file
	fi, _ := osFile.Stat()

	// If the file is a directory, return 404 as cache file doesn't exist for directory
	if fi.IsDir() || isSymlinkDir(fi, fullPath) {
		debug(2, "Cache for directory not available")
		http.NotFound(writer, request)
		service.debugInfo.requestServed(int64(0))
		logHttp(service, request, 404, 0)
		return
	}

	// we use for etag the sha1sum of the full path followed the mtime
	mtime := fi.ModTime().UTC().Format(http.TimeFormat)
	etag := `"` + sha1string(path+mtime) + `"`
	inm := request.Header.Get("If-None-Match")
	if inm == etag {
		debug(4, "If-None-Match match found for %s", etag)
		writer.WriteHeader(http.StatusNotModified)
		logHttp(service, request, 304, 0)
	} else {
		writer.Header().Set("Last-Modified", mtime)
		writer.Header().Set("ETag", etag)
		writer.Header().Set("Cache-Control", "max-age=0, private, must-revalidate")
		debug(4, "Etag sent: %s", etag)

		http.ServeContent(writer, request, thumbnailPath, fi.ModTime(), osFile)
		service.debugInfo.requestServed(fi.Size())
		logHttp(service, request, 200, int(fi.Size()))
	}

	return

}

func (service *MercuryFsService) serveShares(writer http.ResponseWriter, request *http.Request) {
	var user *HdaUser
	if !isAdmin(request) {
		user = service.checkAuthHeader(writer, request)
		if user == nil {
			return
		}
	}
	var shares []*HdaShare
	var err error
	if service.Shares.rootDir == "" && !(user == nil || user.IsDemo) {
		shares, err = user.AvailableShares()
		if err != nil {
			http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
		}
	} else {
		service.Shares.updateShares()
		shares = service.Shares.Shares
	}
	debug(5, "========= DEBUG Share request: %d", len(shares))
	json := SharesJson(shares)
	debug(5, "Share JSON: %s", json)
	etag := `"` + sha1bytes([]byte(json)) + `"`
	inm := request.Header.Get("If-None-Match")
	if inm == etag {
		debug(4, "If-None-Match match found for %s", etag)
		writer.WriteHeader(http.StatusNotModified)
		service.debugInfo.requestServed(int64(0))
		logHttp(service, request, http.StatusNotModified, 0)
	} else {
		debug(4, "If-None-Match (%s) match NOT found for Etag %s", inm, etag)
		size := int64(len(json))
		writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		writer.Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
		writer.Header().Set("ETag", etag)
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "max-age=0, private, must-revalidate")
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte(json))
		service.debugInfo.requestServed(size)
		logHttp(service, request, http.StatusOK, int(size))
	}
}

func GetLocalAddr(rootDir string) (string, error) {

	if rootDir != "" {
		return "127.0.0.1", nil
	}

	dbconn, err := sql.Open("mysql", MYSQL_CREDENTIALS)
	if err != nil {
		logError(err.Error())
		return "", err
	}
	defer dbconn.Close()

	var prefix, addr string
	q := "SELECT value FROM settings WHERE name = ?"
	row := dbconn.QueryRow(q, "net")
	err = row.Scan(&prefix)
	if err != nil {
		logInfo(err.Error())
		return "", err
	}
	row = dbconn.QueryRow(q, "self-address")
	err = row.Scan(&addr)
	if err != nil {
		logError("Error scanning self-address: %s\n", err.Error())
		return "", err
	}

	debug(4, "prefix: %s\taddr: %s", prefix, addr)
	return prefix + "." + addr, nil
}

func pathForLog(u *url.URL) string {
	var buf bytes.Buffer
	buf.WriteString(u.Path)
	if u.RawQuery != "" {
		buf.WriteByte('?')
		buf.WriteString(u.RawQuery)
	}
	if u.Fragment != "" {
		buf.WriteByte('#')
		buf.WriteString(url.QueryEscape(u.Fragment))
	}
	return buf.String()
}

func isSymlinkDir(m os.FileInfo, fullPath string) bool {
	// debug(1, "isSymlinkDir(%s)", m.Name())
	// not a symlink, so return
	if m.Mode()&os.ModeSymlink == 0 {
		// debug(1, "isSymlink: not a symlink")
		return false
	}
	// it's a symlink, is the destination a directory?
	linkedPath := fullPath + "/" + m.Name()
	// debug(1, "isSymlink: %s - %s / %s", fullPath, filePath.Dir(fullPath), m.Name())
	link, err := os.Readlink(linkedPath)
	if err != nil {
		// debug(1, "isSymlink: error reading symlink: %s", err)
		return false
	}
	// default to absolute path
	dest := link
	if link[0] != '/' {
		// if not starting in /, it's a relative path
		dest = fullPath + "/" + link
	}
	// debug(1, "isSymlink: symlink is: %s", dest)
	file, err := os.Open(dest)
	if err != nil {
		// debug(1, "isSymlink: error opening symlink destination: %s", err)
		return false
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		// debug(1, "isSymlink: error in stat: %s", err)
		return false
	}
	// debug(1, "isSymlink: info: %s", fi)
	return fi.IsDir()
}

func (service *MercuryFsService) appsList(writer http.ResponseWriter, request *http.Request) {
	apps, err := newHdaApps()
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	service.Apps = apps
	service.Apps.list()
	debug(5, "========= DEBUG apps_list request: %d", len(service.Shares.Shares))
	json := service.Apps.toJson()
	debug(5, "App JSON: %s", json)
	etag := `"` + sha1bytes([]byte(json)) + `"`
	inm := request.Header.Get("If-None-Match")
	if inm == etag {
		debug(4, "If-None-Match match found for %s", etag)
		writer.WriteHeader(http.StatusNotModified)
		service.debugInfo.requestServed(int64(0))
		logHttp(service, request, http.StatusNotModified, 0)
	} else {
		debug(4, "If-None-Match (%s) match NOT found for Etag %s", inm, etag)
		size := int64(len(json))
		writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		writer.Header().Set("ETag", etag)
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "max-age=0, private, must-revalidate")
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte(json))
		service.debugInfo.requestServed(size)
		logHttp(service, request, http.StatusOK, int(size))
	}
}

func (service *MercuryFsService) getMetadata(writer http.ResponseWriter, request *http.Request) {
	// get the filename and the hint
	q := request.URL

	filename, err := url.QueryUnescape(q.Query().Get("f"))
	if err != nil {
		debug(3, "get_metadata error parsing file: %s", err)
		http.NotFound(writer, request)
		logHttp(service, request, 404, 0)
		return
	}
	hint, err := url.QueryUnescape(q.Query().Get("h"))
	if err != nil {
		debug(3, "get_metadata error parsing hint: %s", err)
		http.NotFound(writer, request)
		logHttp(service, request, 404, 0)
		return
	}
	debug(5, "metadata filename: %s", filename)
	debug(5, "metadata hint: %s", hint)
	// FIXME
	json, err := service.metadata.GetMetadata(filename, hint)
	if err != nil {
		debug(3, "metadata error: %s", err)
		http.NotFound(writer, request)
		logHttp(service, request, 404, 0)
		return
	}
	debug(5, "========= DEBUG get_metadata request: %d", len(service.Shares.Shares))
	debug(5, "metadata JSON: %s", json)
	etag := `"` + sha1bytes([]byte(json)) + `"`
	inm := request.Header.Get("If-None-Match")
	if inm == etag {
		debug(4, "If-None-Match match found for %s", etag)
		writer.WriteHeader(http.StatusNotModified)
		service.debugInfo.requestServed(int64(0))
		logHttp(service, request, http.StatusUnauthorized, 0)
	} else {
		debug(4, "If-None-Match (%s) match NOT found for Etag %s", inm, etag)
		size := int64(len(json))
		writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		writer.Header().Set("ETag", etag)
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "max-age=0, private, must-revalidate")
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte(json))
		service.debugInfo.requestServed(size)
		logHttp(service, request, http.StatusOK, int(size))
	}
}

func (service *MercuryFsService) printRequest(request *http.Request) {
	debug(5, "REQUEST [from %s] BEGIN =========================", request.RemoteAddr)
	if request.Method != "POST" {
		rawRequest, _ := httputil.DumpRequest(request, true)
		debug(5, "%s", rawRequest)
	} else {
		debug(5, "POST Request to %s (details removed)", request.URL)
	}
	debug(5, "REQUEST END =========================")
}

func (service *MercuryFsService) topVhostFilter(writer http.ResponseWriter, request *http.Request) {
	header := writer.Header()
	ua := request.Header.Get("User-Agent")
	// since data will change with the session, we should indicate that to keep caching!
	header.Add("Vary", "Session")
	if ua == "" {
		service.printRequest(request)
		// if no UA, it's an API call
		service.apiRouter.ServeHTTP(writer, request)
		return
	}

	// search for vhost
	re := regexp.MustCompile(`Vhost/([^\s]*)`)
	matches := re.FindStringSubmatch(ua)
	debug(5, "VHOST matches %q *************************", matches)
	if len(matches) != 2 {
		service.printRequest(request)
		// if no vhost, default to API?
		service.apiRouter.ServeHTTP(writer, request)
		return
	}

	service.printRequest(request)

	vhost := matches[1]
	app := service.Apps.get(vhost)
	if app == nil {
		debug(5, "No matching app found for VHOST %s", vhost)
		http.Error(writer, "Unknown App", http.StatusNotFound)
		return
	}

	debug(5, "VHOST REQUEST FOR %s *************************", vhost)

	request.URL.Host = "hda"
	request.Host = vhost

	// FIXME - support https and other ports later
	remote, err := url.Parse("http://" + vhost)
	if err != nil {
		debug(5, "REQUEST ERROR: %s", err)
		http.NotFound(writer, request)
		return
	}

	// proxy the app request
	proxy := httputil.NewSingleHostReverseProxy(remote)
	// since data will change with the UA, we should indicate that to keep caching!
	header.Add("Vary", "User-Agent")
	proxy.ServeHTTP(writer, request)
}

// delete a file!
func (service *MercuryFsService) deleteFile(writer http.ResponseWriter, request *http.Request) {
	q := request.URL
	path := q.Query().Get("p")
	share := q.Query().Get("s")

	debug(2, "deleteFile DELETE request")

	service.printRequest(request)

	fullPath, err := service.fullPathToFile(share, path)

	// if using the welcome server, just return OK without deleting anything
	if !noDelete {
		if err != nil {
			debug(2, "File not found: %s", err)
			http.NotFound(writer, request)
			service.debugInfo.requestServed(int64(0))
			logHttp(service, request, 404, 0)
			return
		}
		err = os.Remove(fullPath)
		if err != nil {
			debug(2, "Error removing file: %s", err.Error())
			writer.WriteHeader(http.StatusExpectationFailed)
			service.debugInfo.requestServed(int64(0))
			logHttp(service, request, 417, 0)
			return
		}
	} else {
		debug(2, "NOTICE: Running in no-delete mode. Would have deleted: %s", fullPath)
	}

	writer.WriteHeader(http.StatusOK)
	logHttp(service, request, http.StatusOK, 0)

	return
}

// upload a file!
func (service *MercuryFsService) uploadFile(writer http.ResponseWriter, request *http.Request) {
	q := request.URL
	path := q.Query().Get("p")
	share := q.Query().Get("s")

	debug(2, "upload_file POST request")

	// do NOT print the whole request, as an image may be way way too big
	service.printRequest(request)

	// full_path, err := service.fullPathToFile(share, path+"/upload")

	// if using the welcome server, just return OK without deleting anything
	if !noUpload {

		// if err != nil {
		// 	debug(2, "File not found: %s", err)
		// 	http.NotFound(writer, request)
		// 	service.debug_info.requestServed(int64(0))
		// 	log("\"POST %s\" 404 0 \"%s\"", query, ua)
		// 	return
		// }

		// max size is 20MB of memory
		err := request.ParseMultipartForm(32 << 20)

		if err != nil {
			debug(2, "Error parsing image: %s", err.Error())
			writer.WriteHeader(http.StatusPreconditionFailed)
			service.debugInfo.requestServed(int64(0))
			logHttp(service, request, 412, 0)
			return
		}

		// debug(2, "Form data: %s", values)
		file, handler, err := request.FormFile("file")
		if err != nil {
			debug(2, "Error finding uploaded file: %s", err.Error())
			writer.WriteHeader(http.StatusExpectationFailed)
			service.debugInfo.requestServed(int64(0))
			logHttp(service, request, 417, 0)
			return
		}
		defer file.Close()

		fullPath, _ := service.fullPathToFile(share, path+"/"+handler.Filename)
		//check if the file name is valid
		if !validFilename(fullPath) {
			debug(2, "invalid filename")
			writer.WriteHeader(http.StatusUnsupportedMediaType)
			service.debugInfo.requestServed(int64(0))
			logHttp(service, request, 415, 0)
			return
		}

		//check file status
		status := checkFileExists(fullPath, file)
		file.Seek(0, 0)

		var f *os.File
		if status == FILE_NOT_EXISTS {
			//file not exists, create and write it
			f, err = os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, 0644)

		} else if status == FILE_EXISTS {
			//file exists but md5 is different, rename it
			fullPath = renameFile(fullPath)
			f, err = os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, 0644)
		}
		if err != nil {
			debug(2, "Error creating uploaded file: %s", err.Error())
			writer.WriteHeader(http.StatusServiceUnavailable)
			service.debugInfo.requestServed(int64(0))
			logHttp(service, request, 503, 0)
			return
		} // status == FILE_SAME_MD5, ignore it

		defer f.Close()
		io.Copy(f, file)

		debug(2, "POST of a file upload parsed successfully")

	} else {
		debug(2, "NOTICE: Running in no-upload mode.")
	}

	writer.WriteHeader(http.StatusOK)
	logHttp(service, request, 200, 0)

	return
}

func (service *MercuryFsService) serveLogs(writer http.ResponseWriter, request *http.Request) {
	q := request.URL
	amt := q.Query().Get("mode")

	mode := 100 // determines the numbers of lines to serve (from last). -1 will cause serving complete log file

	if n, err := strconv.Atoi(amt); err == nil {
		mode = n
	} else {
		if strings.ToLower(amt) == "all" {
			mode = -1
		}
	}

	osFile, err := os.Open(LOGFILE)
	if err != nil {
		debug(2, "Error opening log file: %s", err.Error())
		http.NotFound(writer, request)
		service.debugInfo.requestServed(int64(0))
		logHttp(service, request, 404, 0)
		return
	}
	defer osFile.Close()

	fi, _ := osFile.Stat()
	if mode == -1 {
		http.ServeContent(writer, request, osFile.Name(), fi.ModTime(), osFile)
		logHttp(service, request, 200, int(fi.Size()))
	} else {
		data, err := Tail(mode)
		if err != nil {
			writer.WriteHeader(http.StatusNotFound)
		}
		size, _ := writer.Write(data)
		logHttp(service, request, 200, size)
	}
}
