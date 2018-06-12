package main

import (
	"sync"
	"database/sql"
	"fmt"
	"time"
	"crypto/rand"
)

type HdaUser struct {
	id            int
	Login         string    `json:"login"`
	Name          string    `json:"name"`
	UpdatedAt     time.Time `json:"updated_at"`
	LastRequestAt time.Time `json:"last_request_at"`
	LastCheckedAt time.Time `json:"last_checked_at"`
}

type HdaUsers struct {
	Users map[string]*HdaUser
	sync.RWMutex
}

func NewHdaUsers() *HdaUsers {
	return &HdaUsers{Users: make(map[string]*HdaUser)}
}

func tokenGenerator() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (users *HdaUsers) queryUser(pin string) (*string, error) {
	dbconn, err := sql.Open("mysql", MYSQL_CREDENTIALS)
	if err != nil {
		log(err.Error())
		return nil, err
	}
	defer dbconn.Close()
	q := "SELECT id, login, name, updated_at FROM users WHERE pin=?"
	user := new(HdaUser)
	err = dbconn.QueryRow(q, pin).Scan(&user.id, &user.Login, &user.Name, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	authToken := users.updateUserIfExists(user)
	if authToken != "" {
		return &authToken, nil
	} else {
		authToken = tokenGenerator()
		user.LastCheckedAt = time.Now()
		users.Lock()
		users.Users[authToken] = user
		users.Unlock()
	}
	return &authToken, nil
}

func (users *HdaUsers) updateUserIfExists(newUser *HdaUser) string {
	for authToken, user := range users.Users {
		if user.id == newUser.id {
			newUser.LastCheckedAt = time.Now()
			users.Lock()
			user = newUser
			users.Unlock()
			return authToken
		}
	}
	return ""
}

func (users *HdaUsers) find(authToken string) *HdaUser {
	users.Lock()
	defer users.Unlock()
	return users.Users[authToken]
}

func (user *HdaUser) AvailableShares() ([]*HdaShare, error) {
	dbconn, err := sql.Open("mysql", MYSQL_CREDENTIALS)
	if err != nil {
		log(err.Error())
		return nil, err
	}
	defer dbconn.Close()
	q := "SELECT s.id, s.name, s.updated_at, s.path, s.tags, " +
		"CASE WHEN cw.id IS NULL THEN 'false' ELSE 'true' END AS writable " +
		"FROM cap_accesses as ca " +
		"INNER JOIN shares AS s ON s.id = ca.share_id " +
		"INNER JOIN users AS u ON u.id = ca.user_id " +
		"LEFT JOIN cap_writers AS cw ON ca.user_id = cw.user_id AND ca.share_id = cw.share_id " +
		"WHERE u.id = ? AND s.visible = 1 ORDER BY s.name ASC;"
	rows, err := dbconn.Query(q, user.id)
	if err != nil {
		log(err.Error())
		return nil, err
	}
	newShares := make([]*HdaShare, 0)
	for rows.Next() {
		share := new(HdaShare)
		rows.Scan(&share.Name, &share.UpdatedAt, &share.Path, &share.Tags, &share.IsWritable)
		newShares = append(newShares, share)
	}
	return newShares, nil
}

func (user *HdaUser) HasReadAccess(shareName string) (access bool, err error) {
	dbconn, err := sql.Open("mysql", MYSQL_CREDENTIALS)
	if err != nil {
		log(err.Error())
		return
	}
	defer dbconn.Close()
	q := "SELECT EXISTS(SELECT 1 from cap_accesses as ca " +
		"INNER JOIN shares AS s on s.id = ca.share_id " +
		"INNER JOIN users AS u ON u.id = ca.user_id " +
		"WHERE u.id = ? AND s.name = ? AND s.visible = 1);"
	err = dbconn.QueryRow(q, user.id, shareName).Scan(&access)
	return
}

func (user *HdaUser) HasWriteAccess(shareName string) (access bool, err error) {
	dbconn, err := sql.Open("mysql", MYSQL_CREDENTIALS)
	if err != nil {
		log(err.Error())
		return
	}
	defer dbconn.Close()
	q := "SELECT EXISTS(SELECT 1 from cap_writers as ca " +
		"INNER JOIN shares AS s on s.id = ca.share_id " +
		"INNER JOIN users AS u ON u.id = ca.user_id " +
		"WHERE u.id = ? AND s.name = ? AND s.visible = 1);"
	err = dbconn.QueryRow(q, user.id, shareName).Scan(&access)
	return
}
