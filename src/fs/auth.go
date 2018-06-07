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

func (service *MercuryFsService) authenticate(writer http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	data := make(map[string]interface{})
	err := decoder.Decode(&data)
	if err != nil {
		panic(err)
	}
	defer request.Body.Close()
	pin, ok := data["pin"].(string)
	if !ok {
		// pin is not a string
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	authToken, err := service.Users.queryUser(pin)
	switch {
	case err == sql.ErrNoRows:
		log("No user with pin: %s", pin)
		http.Error(writer, "Authentication Failed", http.StatusUnauthorized)
		break
	case err != nil:
		http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
		log(err.Error())
		break
	default:
		respJson := fmt.Sprintf("{\"auth_token\": \"%s\"}", *authToken)
		writer.WriteHeader(http.StatusOK)
		size := int64(len(respJson))
		writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		writer.Header().Set("Content-Type", "application/json")
		writer.Write([]byte(respJson))
	}
}

func (service *MercuryFsService) checkAuthHeader(w http.ResponseWriter, r *http.Request) (user *HdaUser) {
	authToken := r.Header.Get("Authorization")
	user = service.Users.find(authToken)
	if user == nil {
		http.Error(w, "Authentication Failed", http.StatusUnauthorized)
	}
	return
}

func (service *MercuryFsService) authMiddleware(pass http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service.checkAuthHeader(w, r)
		pass(w, r)
	}
}

func (service *MercuryFsService) shareReadAccess(pass http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := service.checkAuthHeader(w, r)
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

func (service *MercuryFsService) shareWriteAccess(pass http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := service.checkAuthHeader(w, r)
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
