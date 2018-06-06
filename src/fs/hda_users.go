package main

import (
	"sync"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	"crypto/rand"
)

type HdaUser struct {
	Id            int       `json:"id"`
	Login         string    `json:"login"`
	Name          string    `json:"name"`
	SessionToken  string    `json:"-"`
	UpdatedAt     time.Time `json:"updated_at"`
	LastRequestAt time.Time `json:"last_request_at"`
	LastCheckedAt time.Time `json:"last_checked_at"`
}

type HdaUsers struct {
	Users []*HdaUser `json:"users"`
	sync.RWMutex
}

func NewHdaUsers() *HdaUsers {
	return &HdaUsers{Users: make([]*HdaUser, 0)}
}

func tokenGenerator() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (this *HdaUsers) queryUser(pin string) (*HdaUser, error) {
	dbconn, err := sql.Open("mysql", MYSQL_CREDENTIALS)
	if err != nil {
		log(err.Error())
		return nil, err
	}
	defer dbconn.Close()
	q := "SELECT id, login, name, updated_at FROM users WHERE pin=?"
	user := new(HdaUser)
	err = dbconn.QueryRow(q, pin).Scan(&user.Id, &user.Login, &user.Name, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if isUpdated := this.updateUserIfExists(user); isUpdated {
		return user, nil
	} else {
		user.SessionToken = tokenGenerator()
		user.LastCheckedAt = time.Now()
		this.Lock()
		this.Users = append(this.Users, user)
		this.Unlock()
	}

	return user, nil
}

func (this *HdaUsers) updateUserIfExists(newUser *HdaUser) (updated bool) {
	updated = false
	for i := range this.Users {
		if oldUser := this.Users[i]; oldUser.Id == newUser.Id {
			newUser.SessionToken = oldUser.SessionToken
			newUser.LastCheckedAt = time.Now()
			this.Lock()
			this.Users[i] = newUser
			this.Unlock()
			return true
		}
	}
	return
}

func (this *HdaUsers) get(token string) *HdaUser {
	for i := range this.Users {
		if this.Users[i].SessionToken == token {
			return this.Users[i]
		}
	}
	return nil
}

func (this *HdaUsers) toJson() string {
	b, err := json.Marshal(this)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return string(b)
}
