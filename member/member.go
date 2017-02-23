package member

import (
	"log"
	"database/sql"
	"github.com/vvanpo/makerspace/talk"
	"github.com/vvanpo/makerspace/billing"
	"time"
)

type Member struct {
	Id              int
	Username        string
	Name            string
	Email           string
	Agreed_to_terms bool
	Registered      time.Time
	gratuitous		bool
	*Admin
	*Student
	password_key    string
	password_salt   string
	talk            *talk.Talk_user
	membership		*billing.Invoice
	*Members
	payment         *billing.Profile
}

//TODO: support null password keys, and use e-mail verification for login
//TODO: check corporate account
func (ms *Members) Get_member_by_username(username string) *Member {
	m := &Member{Username: username, Members: ms}
	var password_key, password_salt sql.NullString
	if err := m.QueryRow(
		"SELECT"+
		"	id, "+
		"	name, "+
		"	password_key, "+
		"	password_salt, "+
		"	email, "+
		"	agreed_to_terms, "+
		"	registered, "+
		"	gratuitous "+
		"FROM member "+
		"WHERE username = $1",
		username).Scan(&m.Id, &m.Name, &password_key, &password_salt, &m.Email,
		&m.Agreed_to_terms, &m.Registered, &m.gratuitous);
		err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	m.password_key = password_key.String
	m.password_salt = password_salt.String
	m.get_student()
	m.get_admin()
	m.get_membership()
	return m
}

func (ms *Members) Get_member_by_id(id int) *Member {
	m := &Member{Id: id, Members: ms}
	var password_key, password_salt sql.NullString
	if err := m.QueryRow(
		"SELECT"+
		"	username, "+
		"	name, "+
		"	password_key, "+
		"	password_salt, "+
		"	email, "+
		"	agreed_to_terms, "+
		"	registered, "+
		"	gratuitous "+
		"FROM member "+
		"WHERE id = $1",
		id).Scan(&m.Username, &m.Name, &password_key, &password_salt, &m.Email,
		&m.Agreed_to_terms, &m.Registered, &m.gratuitous);
		err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	m.password_key = password_key.String
	m.password_salt = password_salt.String
	m.get_student()
	m.get_admin()
	m.get_membership()
	return m
}

func (m *Member) Authenticate(password string) bool {
	if m.password_key == key(password, m.password_salt) {
		return true
	}
	return false
}

//TODO
func (m *Member) Active() bool {
	if m.gratuitous || m.membership != nil {
		return true
	}
	return false
}

func (m *Member) Change_password(password string) {
	m.password_salt = Rand256()
	m.password_key = key(password, m.password_salt)
	if _, err := m.Exec("UPDATE member (password_key, password_salt) SET "+
		"password_key = $1, password_salt = $2 WHERE id = $3",
		m.password_key, m.password_salt, m.Id); err != nil {
		log.Panic(err)
	}
}

//TODO: forgotten password reset by e-mail

func (m *Member) Talk_user() *talk.Talk_user {
	if m.talk == nil {
		m.talk = m.Talk_api.Get_user(m.Id)
	}
	return m.talk
}

func (m *Member) Payment() *billing.Profile {
	if m.payment == nil {
		m.payment = m.Get_profile(m.Id)
	}
	return m.payment
}

func (m *Member) get_membership() {
	if m.Payment() == nil {
		return
	}
	m.membership = m.payment.Get_membership()
}

func (m *Member) New_membership() {
	if m.Payment() == nil {
		m.payment = m.New_profile(m.Id)
	}
	if m.Student != nil {
		m.membership = m.payment.New_membership(true)
		return
	}
	m.membership = m.payment.New_membership(false)
}
