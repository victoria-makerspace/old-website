package member

import (
	"database/sql"
	"github.com/vvanpo/makerspace/billing"
	"github.com/vvanpo/makerspace/talk"
	"github.com/lib/pq"
	"log"
	"time"
)

type Member struct {
	Id              int
	Username        string
	Name            string
	Email           string
	Avatar_url		string
	Telephone       string
	Agreed_to_terms bool
	Registered      time.Time
	Gratuitous      bool
	Approved		bool
	*Admin
	*Student
	*Members
	Membership_invoice    *billing.Invoice
	password_key  string
	password_salt string
	talk          *talk.Talk_user
	payment       *billing.Profile
}

//TODO: support null password keys, and use e-mail verification for login
//TODO: check corporate account
func (ms *Members) Get_member_by_id(id int) *Member {
	m := &Member{Id: id, Members: ms}
	var email, password_key, password_salt, avatar_url sql.NullString
	var approved_at pq.NullTime
	if err := m.QueryRow(
		"SELECT"+
			"	username,"+
			"	name,"+
			"	password_key,"+
			"	password_salt,"+
			"	email,"+
			"	avatar_url,"+
			"	agreed_to_terms,"+
			"	registered,"+
			"	gratuitous,"+
			"	approved_at "+
			"FROM member "+
			"WHERE id = $1",
		id).Scan(&m.Username, &m.Name, &password_key, &password_salt, &email,
		&avatar_url, &m.Agreed_to_terms, &m.Registered, &m.Gratuitous,
		&approved_at);
		err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	m.password_key = password_key.String
	m.password_salt = password_salt.String
	m.Email = email.String
	m.Avatar_url = avatar_url.String
	if approved_at.Valid {
		m.Approved = true
	}
	m.get_student()
	m.get_admin()
	if m.Payment() != nil {
		m.Membership_invoice = m.payment.Get_membership()
	}
	if m.Avatar_url == "" {
		go func() {
			if t := m.Talk_user(); t != nil {
				m.set_avatar_url(t.Avatar_url())
			}
		}()
	}
	return m
}

func (ms *Members) Get_member_by_username(username string) *Member {
	var member_id int
	if err := ms.QueryRow(
		"SELECT id "+
			"FROM member "+
			"WHERE username = $1",
		username).Scan(&member_id);
		err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	return ms.Get_member_by_id(member_id)
}

//TODO: cascade through all tables
func (m *Member) Delete_member() {
	if _, err := m.Exec("DELETE FROM member WHERE id = $1", m.Id); err != nil {
		log.Panic(err)
	}
}

func (m *Member) Verified_email() bool {
	if m.Email != "" {
		return true
	}
	return false
}

func (m *Member) Authenticate(password string) bool {
	if m.password_key == key(password, m.password_salt) {
		return true
	}
	return false
}

func (m *Member) Set_password(password string) {
	m.password_salt = Rand256()
	m.password_key = key(password, m.password_salt)
	if _, err := m.Exec("UPDATE member "+
		"SET password_key = $1, password_salt = $2 "+
		"WHERE id = $3",
		m.password_key, m.password_salt, m.Id); err != nil {
		log.Panic(err)
	}
	if _, err := m.Exec("DELETE FROM reset_password_token "+
		"WHERE member = $1", m.Id); err != nil {
		log.Panic(err)
	}
}

func (m *Member) set_email(email string) {
	m.Email = email
	if _, err := m.Exec("UPDATE member "+
		"SET email = $1 "+
		"WHERE id = $2", email, m.Id); err != nil {
		log.Panic(err)
	}
}

func (m *Member) set_avatar_url(avatar_url string) {
	m.Avatar_url = avatar_url
	if _, err := m.Exec("UPDATE member "+
		"SET avatar_url = $1 "+
		"WHERE id = $2", avatar_url, m.Id); err != nil {
		log.Panic(err)
	}
}

func (m *Member) Send_password_reset() {
	if !m.Verified_email() {
		return
	}
	token := Rand256()
	if _, err := m.Exec("INSERT INTO reset_password_token (member, token) "+
		"VALUES ($1, $2) "+
		"ON CONFLICT (member) DO UPDATE SET"+
		"	(token, time) = ($2, now())", m.Id, token); err != nil {
		log.Panic("Failed to set password reset token: ", err)
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

func (m *Member) Send_email_verification(email string) {
	token := Rand256()
	if _, err := m.Exec("INSERT INTO email_verification_token"+
		"	(member, email, token) "+
		"VALUES ($1, $2, $3) "+
		"ON CONFLICT (member) DO UPDATE SET"+
		"	(email, token, time) = ($2, $3, now())", m.Id, email, token); err != nil {
		log.Panic("Failed to set email verification token: ", err)
	}
	msg := message{subject: "Makerspace.ca: e-mail verification"}
	msg.set_from("Makerspace", "admin@makerspace.ca")
	msg.add_to(m.Name, email)
	//TODO use config.json value for domain
	msg.body = "Hello " + m.Name + ",\n\n" +
		"Please verify your e-mail address (" + email + ") by visiting " +
		"https://devel.makerspace.ca/sso/verify-email?token=" + token + "\n\n"
	m.send_email("admin@makerspace.ca", msg.emails(), msg.format())
}

func (ms *Members) Verify_email(token string) bool {
	m, email := ms.get_member_from_verification_token(token)
	if m == nil {
		return false
	}
	m.talk = m.Sync(m.Id, m.Username, email, m.Name)
	if m.talk == nil {
		log.Panicf("Invalid talk user: (%d) %s <%s>\n", m.Id, m.Username,
			email)
	}
	if !m.Verified_email() {
		m.talk.Activate()
	}
	m.set_email(email)
	//TODO: delete unverified members with this pending verification
	if _, err := m.Exec("DELETE FROM email_verification_token "+
		"WHERE email = $1", email); err != nil {
		log.Panic(err)
	}
	return true
}

func (m *Member) Talk_user() *talk.Talk_user {
	if !m.Verified_email() {
		return nil
	} else if m.talk == nil {
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

func (m *Member) New_membership_invoice() {
	if m.Payment() == nil {
		m.payment = m.New_profile(m.Id)
	}
	m.Membership_invoice = m.payment.New_pending_membership(m.Student != nil)
}

func (m *Member) Cancel_membership() {
	if _, err := m.Exec(
		"UPDATE member "+
		"SET"+
		"	gratuitous = 'f',"+
		"	approved_at = NULL,"+
		"	approved_by = NULL "+
		"WHERE id = $1", m.Id); err != nil {
		log.Panic(err)
	}
	m.Gratuitous = false
	m.Approved = false
	if m.Membership_invoice != nil {
		m.payment.Cancel_membership()
	}
	m.Talk_user().Remove_from_group("Members")
}

func (m *Member) Approved_on() time.Time {
	var approved_at time.Time
	if !m.Approved {
		return approved_at
	}
	if err := m.QueryRow(
		"SELECT approved_at "+
		"FROM member "+
		"WHERE id = $1", m.Id).Scan(&approved_at); err != nil {
		log.Panic(err)
	}
	return approved_at
}

func (m *Member) Approved_by() *Member {
	var approved_by int
	if !m.Approved {
		return nil
	}
	if err := m.QueryRow(
		"SELECT approved_by "+
		"FROM member "+
		"WHERE id = $1", m.Id).Scan(&approved_by); err != nil {
		log.Panic(err)
	}
	return m.Get_member_by_id(approved_by)
}
