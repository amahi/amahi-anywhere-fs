package main

import (
	"strings"
)

func getEscapedQueryParam(query string, paramName string) string {
	for query != "" {
		key := query
		if i := strings.IndexAny(key, "&;"); i >= 0 {
			key, query = key[:i], key[i+1:]
		} else {
			query = ""
		}
		if key == "" {
			continue
		}
		value := ""
		if i := strings.Index(key, "="); i >= 0 {
			key, value = key[:i], key[i+1:]
		}

		if key == paramName {
			return value
		}
	}
	return ""
}
