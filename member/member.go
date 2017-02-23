package member

import (
	"log"
	"github.com/vvanpo/makerspace/talk"
	"time"
)

type Member struct {
	Id              int
	Username        string
	Name            string
	Email           string
	Active          bool
	Agreed_to_terms bool
	Registered      time.Time
	Admin           bool
	Student         bool
	Corporate       bool //TODO
	password_key    string
	password_salt   string
	talk            *talk.Talk_user
	*Members
}

func (m *Member) Authenticate(password string) bool {
	if m.password_key == key(password, m.password_salt) {
		return true
	}
	return false
}

func (m *Member) Change_password(password string) {
	m.password_salt = Rand256()
	m.password_key = key(password, m.password_salt)
	if _, err := m.db.Exec("UPDATE member (password_key, password_salt) SET "+
		"password_key = $1, password_salt = $2 WHERE username = $3",
		m.password_key, m.password_salt, m.Username); err != nil {
		log.Panic(err)
	}
}

//TODO: forgotten password reset by e-mail

func (m *Member) Talk_user() *talk.Talk_user {
	if m.talk == nil {
		m.talk = m.talk_api.Get_user(m.Id)
	}
	return m.talk
}
