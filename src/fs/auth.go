package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func use(h http.HandlerFunc, middleware ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, m := range middleware {
		h = m(h)
	}
	return h
}

func parseAuthToken(r *http.Request) (authToken string) {
	authToken = r.Header.Get("Authorization")
	if authToken == "" {
		authToken = r.URL.Query().Get("auth")
	}
	return
}

func isAdmin(r *http.Request) bool {
	// if Authorization header is not present, this is admin user
	return parseAuthToken(r) == ""
}

func (service *MercuryFsService) authenticate(writer http.ResponseWriter, request *http.Request) {
	// decode and parse json request body
	defer request.Body.Close()
	decoder := json.NewDecoder(request.Body)
	data := make(map[string]interface{})
	err := decoder.Decode(&data)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		logHttp(request, http.StatusBadRequest, 0)
		return
	}
	// read pin from the json body
	pin, ok := data["pin"].(string)
	if !ok {
		// pin is not a string, send 400 Bad Request
		writer.WriteHeader(http.StatusBadRequest)
		logHttp(request, http.StatusBadRequest, 0)
		return
	}
	// query user for the given pin from the list of all users
	authToken, err := service.Users.queryUser(pin)
	switch {
	case err == sql.ErrNoRows: // if no such user exits, send 401 Unauthorized
		logInfo("No user with pin: %s", pin)
		errMsg := "Authentication Failed"
		http.Error(writer, errMsg, http.StatusUnauthorized)
		logHttp(request, http.StatusUnauthorized, len(errMsg))
		break
	case err != nil: // if some other error, send 500 Internal Server Error
		logError(err.Error())
		errMsg := "Internal Server Error"
		http.Error(writer, errMsg, http.StatusInternalServerError)
		logHttp(request, http.StatusInternalServerError, len(errMsg))
		break
	default: // if no error, send proper auth token for that user
		respJson := fmt.Sprintf("{\"auth_token\": \"%s\"}", *authToken)
		writer.WriteHeader(http.StatusOK)
		size := int64(len(respJson))
		writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		writer.Header().Set("Content-Type", "application/json")
		writer.Write([]byte(respJson))
		logHttp(request, http.StatusOK, int(size))
	}
}

func (service *MercuryFsService) logout(writer http.ResponseWriter, request *http.Request) {
	authToken := parseAuthToken(request)
	service.Users.remove(authToken)
	writer.WriteHeader(http.StatusOK)
	logHttp(request, http.StatusOK, 0)
}

func (service *MercuryFsService) checkAuthHeader(w http.ResponseWriter, r *http.Request) (user *HdaUser) {
	authToken := parseAuthToken(r)
	user = service.Users.find(authToken)
	// if user is nil, respond with 401 Unauthorized
	if user == nil {
		errMsg := "Authentication Failed"
		http.Error(w, errMsg, http.StatusUnauthorized)
		logHttp(r, http.StatusUnauthorized, len(errMsg))
	}
	return user
}

func (service *MercuryFsService) authMiddleware(pass http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAdmin(r) {
			// auth header is not present, pass as this is admin user
			pass(w, r)
		} else {
			// auth header is present, pass only if a user exists for the given auth_token
			user := service.checkAuthHeader(w, r)
			if user != nil {
				pass(w, r)
			}
		}
	}
}

func (service *MercuryFsService) shareReadAccess(pass http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAdmin(r) {
			// auth header is not present, pass as this is admin user
			pass(w, r)
		} else {
			user := service.checkAuthHeader(w, r)
			// if user is nil, we have already responded with 401 Unauthorized, so return
			if user == nil {
				return
			}
			// check for share name, and if the user has read access for it
			// if no access, send 403 Forbidden
			// else if error, send 500 Internal Server Error
			shareName := r.URL.Query().Get("s")
			if access, err := user.HasReadAccess(shareName); !access {
				if err == nil {
					errMsg := "Access Forbidden"
					http.Error(w, errMsg, http.StatusForbidden)
					logHttp(r, http.StatusForbidden, len(errMsg))
				} else {
					errMsg := "Internal Server Error"
					http.Error(w, errMsg, http.StatusInternalServerError)
					logHttp(r, http.StatusInternalServerError, len(errMsg))
				}
				return
			}
			pass(w, r)
		}
	}
}

func (service *MercuryFsService) restrictCache(pass http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		share := r.URL.Query().Get("s")
		path := r.URL.Query().Get("p")
		fullPath, _ := service.fullPathToFile(share, path)

		if strings.Contains(fullPath, ".fscache") {
			errMsg := "Cannot access cache via /files"
			http.Error(w, errMsg, http.StatusForbidden)
			logHttp(r, http.StatusForbidden, len(errMsg))
			return
		}

		pass(w, r)
	}
}

func (service *MercuryFsService) shareWriteAccess(pass http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAdmin(r) {
			// auth header is not present, pass as this is admin user
			pass(w, r)
		} else {
			user := service.checkAuthHeader(w, r)
			// if user is nil, we have already responded with 401 Unauthorized, so return
			if user == nil {
				return
			}
			// check for share name, and if the user has write access for it
			// if no access, send 403 Forbidden
			// else if error, send 500 Internal Server Error
			shareName := r.URL.Query().Get("s")
			if access, err := user.HasWriteAccess(shareName); !access {
				if err == nil {
					errMsg := "Access Forbidden"
					http.Error(w, errMsg, http.StatusForbidden)
					logHttp(r, http.StatusForbidden, len(errMsg))
				} else {
					errMsg := "Internal Server Error"
					http.Error(w, errMsg, http.StatusInternalServerError)
					logHttp(r, http.StatusForbidden, len(errMsg))
				}
				return
			}
			pass(w, r)
		}
	}
}
