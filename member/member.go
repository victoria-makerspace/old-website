package member

import (
	"database/sql"
	"github.com/vvanpo/makerspace/billing"
	"github.com/vvanpo/makerspace/talk"
	"log"
	"time"
)

type Member struct {
	Id              int
	Username        string
	Name            string
	Email           string
	Telephone       string
	Agreed_to_terms bool
	Registered      time.Time
	Activated       bool
	*Admin
	*Student
	*Members
	Gratuitous    bool
	password_key  string
	password_salt string
	talk          *talk.Talk_user
	Membership    *billing.Invoice
	payment       *billing.Profile
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
			"	activated, "+
			"	agreed_to_terms, "+
			"	registered, "+
			"	gratuitous "+
			"FROM member "+
			"WHERE username = $1",
		username).Scan(&m.Id, &m.Name, &password_key, &password_salt, &m.Email,
		&m.Activated, &m.Agreed_to_terms, &m.Registered, &m.Gratuitous); err != nil {
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
			"	activated, "+
			"	agreed_to_terms, "+
			"	registered, "+
			"	gratuitous "+
			"FROM member "+
			"WHERE id = $1",
		id).Scan(&m.Username, &m.Name, &password_key, &password_salt, &m.Email,
		&m.Activated, &m.Agreed_to_terms, &m.Registered, &m.Gratuitous); err != nil {
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

func (m *Member) Delete_member() {
	if _, err := m.Exec("DELETE FROM member WHERE id = $1", m.Id); err != nil {
		log.Panic(err)
	}
}

func (m *Member) Activate() {
	if _, err := m.Exec("UPDATE member SET activated = 'true' WHERE id = $1",
		m.Id); err != nil {
		log.Panic(err)
	}
	m.Activated = true
}

func (m *Member) Authenticate(password string) bool {
	if m.password_key == key(password, m.password_salt) {
		return true
	}
	return false
}

//TODO
func (m *Member) Active() bool {
	if m.Gratuitous || m.Membership != nil {
		return true
	}
	return false
}

func (m *Member) Change_password(password string) {
	m.password_salt = Rand256()
	m.password_key = key(password, m.password_salt)
	if _, err := m.Exec("UPDATE member "+
		"SET password_key = $1, password_salt = $2 "+
		"WHERE id = $3",
		m.password_key, m.password_salt, m.Id); err != nil {
		log.Panic(err)
	}
}

//TODO: forgotten password reset by e-mail
func (m *Member) Send_password_reset() {
	token := Rand256()
	if _, err := m.Exec("INSERT INTO reset_password_token (member, token) "+
		"VALUES ($1, $2) "+
		"ON CONFLICT (member) DO UPDATE SET"+
		"	(token, time) = ($2, now())", m.Id, token); err != nil {
		log.Panic("Failed password reset: ", err)
	}
	msg := message{subject: "Makerspace.ca: password reset"}
	msg.set_from("Makerspace", "admin@makerspace.ca")
	msg.add_to(m.Name, m.Email)
	//TODO use config.json value for domain
	msg.body = "Hello " + m.Name + ",\n\n" +
		"Please reset your makerspace password by visiting " +
		"https://devel.makerspace.ca/sso/reset?token=" + token + "\n\n"
	m.send_email("admin@makerspace.ca", msg.emails(), msg.format())
}

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
	m.Membership = m.payment.Get_membership()
}

func (m *Member) New_membership() {
	if m.Payment() == nil {
		m.payment = m.New_profile(m.Id)
	}
	if m.Student != nil {
		m.Membership = m.payment.New_membership(true)
		return
	}
	if m.Membership = m.payment.New_membership(false); m.Membership != nil {
		m.Talk_user().Add_to_group("Members")
	}
}

func (m *Member) Cancel_membership() {
	if m.Gratuitous {
		if _, err := m.Exec(
			"UPDATE member "+
				"SET gratuitous = 'f' "+
				"WHERE id = $1", m.Id); err != nil {
			log.Panic(err)
		}
	}
	m.Gratuitous = false
	if m.Membership != nil {
		m.payment.Cancel_membership()
	}
}
