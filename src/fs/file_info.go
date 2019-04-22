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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type fileInfo struct {
	name     string
	mimeType string
	mtime    time.Time
	size     int64
}

type fileSorter struct {
	files []fileInfo
}

// Len is part of sort.Interface
func (fi *fileSorter) Len() int {
	return len(fi.files)
}

// Swap is part of sort.Interface.
func (fi *fileSorter) Swap(i, j int) {
	fi.files[i], fi.files[j] = fi.files[j], fi.files[i]
}

// Less is part of sort.Interface.
func (fi *fileSorter) Less(i, j int) bool {
	return strings.ToLower(fi.files[i].name) < strings.ToLower(fi.files[j].name)
}

func (f *fileInfo) toJson() string {
	name, _ := json.Marshal(f.name)
	return fmt.Sprintf(`{"name": %s, "mime_type": "%s", "mtime": "%s", "size": %d}`,
		string(name), f.mimeType, f.mtime.Format(http.TimeFormat), f.size)
}

func directoryFileInfos(fis []os.FileInfo, fullPath string) []fileInfo {
	fileInfos := make([]fileInfo, 0)
	for i := range fis {
		if fis[i].Name()[0] == '.' {
			continue
		}
		fileInfo := fileInfo{
			name:  fis[i].Name(),
			mtime: fis[i].ModTime(),
		}
		if fis[i].IsDir() || isSymlinkDir(fis[i], fullPath) {
			fileInfo.mimeType = "text/directory"
			fileInfo.size = 0
		} else {
			fileInfo.mimeType = getContentType(fis[i].Name())
			fileInfo.size = fis[i].Size()
		}
		fileInfos = append(fileInfos, fileInfo)
	}

	sorter := &fileSorter{files: fileInfos}

	sort.Sort(sorter)

	return fileInfos
}

func dirToJSON(osFile *os.File, fullPath string) (string, error) {
	fis, err := osFile.Readdir(0)
	if err != nil {
		return "", err
	}

	fileInfos := directoryFileInfos(fis, fullPath)

	if len(fileInfos) == 0 {
		return "[]", nil
	}

	ss := make([]string, 0)
	for i := range fileInfos {
		temp := fileInfos[i].toJson()
		ss = append(ss, temp)
	}

	result := "[\n"
	result += strings.Join(ss, ",\n ")
	result += "\n]"
	return result, nil
}

func getContentType(fileName string) string {
	encodingMap := map[string]string{
		".pdf":  "application/pdf",
		".ogx":  "application/ogg",
		".anx":  "application/annodex",
		".txt":  "text/plain",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".tif":  "image/tiff",
		".tiff": "image/tiff",
		".png":  "image/png",
		".svg":  "image/svg+xml",
		".mp3":  "audio/mpeg",
		".aac":  "audio/aac",
		".oga":  "audio/ogg",
		".ogg":  "audio/ogg",
		".spx":  "audio/ogg",
		".wav":  "audio/vnd.wave",
		".flac": "audio/flac",
		".axa":  "audio/annodex",
		".m4a":  "audio/mp4",
		".mka":  "audio/x-matroska",
		".axv":  "video/annodex",
		".ogv":  "video/ogg",
		".mov":  "video/quicktime",
		".mkv":  "video/x-matroska",
		".mk3d": "video/x-matroska-3d",
		".mp4":  "video/mp4",
		".m4v":  "video/x-m4v",
		".mpeg": "video/mpeg",
		".mpg":  "video/mpeg",
		".ts":   "video/mpeg",
		".avi":  "video/divx",
		".qt":   "video/quicktime",
		".wmv":  "video/x-ms-wmv",
		".wtv":  "video/x-ms-wtv",
		".flv":  "video/x-flv",
		".3gp":  "video/3gpp",
		".webm": "video/webm",
		".epub": "application/epub+zip",
		".mobi": "application/x-mobipocket",
		".zip":  "application/zip",
		".doc":  "application/msword",
		".dot":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".dotx": "application/vnd.openxmlformats-officedocument.wordprocessingml.template",
		".docm": "application/vnd.ms-word.document.macroEnabled.12",
		".dotm": "application/vnd.ms-word.template.macroEnabled.12",
		".xls":  "application/vnd.ms-excel",
		".xlt":  "application/vnd.ms-excel",
		".xla":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".xltx": "application/vnd.openxmlformats-officedocument.spreadsheetml.template",
		".xlsm": "application/vnd.ms-excel.sheet.macroEnabled.12",
		".xltm": "application/vnd.ms-excel.template.macroEnabled.12",
		".xlam": "application/vnd.ms-excel.addin.macroEnabled.12",
		".xlsb": "application/vnd.ms-excel.sheet.binary.macroEnabled.12",
		".ppt":  "application/vnd.ms-powerpoint",
		".pot":  "application/vnd.ms-powerpoint",
		".pps":  "application/vnd.ms-powerpoint",
		".ppa":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".potx": "application/vnd.openxmlformats-officedocument.presentationml.template",
		".ppsx": "application/vnd.openxmlformats-officedocument.presentationml.slideshow",
		".ppam": "application/vnd.ms-powerpoint.addin.macroEnabled.12",
		".pptm": "application/vnd.ms-powerpoint.presentation.macroEnabled.12",
		".potm": "application/vnd.ms-powerpoint.presentation.macroEnabled.12",
		".ppsm": "application/vnd.ms-powerpoint.slideshow.macroEnabled.12",
		".html": "text/html",
		".htm":  "text/html",
		".csv":  "text/csv",
		// subtitle stuff, with others below
		".srt": "application/x-subrip",
		".sub": "text/vnd.dvb.subtitle",
	}

	subExtensions := []string{".idx", ".sub", ".srt", ".ssa", ".ass", ".smi", ".utf", ".utf8", ".utf-8", ".rt", ".aqt", ".usf", ".jss", ".cdg", ".psb", ".mpsub", ".mpl2", ".pjs", ".dks", ".stl", ".vtt"}
	for _, e := range subExtensions {
		encodingMap[e] = "application/x-subtitle"
	}

	extension := filepath.Ext(fileName)
	result := encodingMap[strings.ToLower(extension)]

	if result == "" {
		result = "application/octet-stream"
	}

	return result
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}
