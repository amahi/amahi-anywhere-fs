package main

import (
	"net/http"
	"encoding/json"
	"database/sql"
	"fmt"
	"strconv"
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
		return
	}
	// read pin from the json body
	pin, ok := data["pin"].(string)
	if !ok {
		// pin is not a string, send 400 Bad Request
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	// query user for the given pin from the list of all users
	authToken, err := service.Users.queryUser(pin)
	switch {
	case err == sql.ErrNoRows: // if no such user exits, send 401 Unauthorized
		log("No user with pin: %s", pin)
		http.Error(writer, "Authentication Failed", http.StatusUnauthorized)
		break
	case err != nil: // if some other error, send 500 Internal Server Error
		http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
		log(err.Error())
		break
	default: // if no error, send proper auth token for that user
		respJson := fmt.Sprintf("{\"auth_token\": \"%s\"}", *authToken)
		writer.WriteHeader(http.StatusOK)
		size := int64(len(respJson))
		writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		writer.Header().Set("Content-Type", "application/json")
		writer.Write([]byte(respJson))
	}
}

func (service *MercuryFsService) logout(w http.ResponseWriter, r *http.Request) {
	authToken := parseAuthToken(r)
	service.Users.remove(authToken)
	w.WriteHeader(http.StatusOK)
}

func (service *MercuryFsService) checkAuthHeader(w http.ResponseWriter, r *http.Request) (user *HdaUser) {
	authToken := parseAuthToken(r)
	user = service.Users.find(authToken)
	// if user is nil, respond with 401 Unauthorized
	if user == nil {
		http.Error(w, "Authentication Failed", http.StatusUnauthorized)
	}
	return
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
					http.Error(w, "Access Forbidden", http.StatusForbidden)
				} else {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
				return
			}
			pass(w, r)
		}
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
					http.Error(w, "Access Forbidden", http.StatusForbidden)
				} else {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
				return
			}
			pass(w, r)
		}
	}
}
