package member

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/stripe/stripe-go"
	"github.com/vvanpo/makerspace/talk"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Member struct {
	Id              int
	Username        string
	Name            string
	Email           string
	Key_card        string
	Telephone       string
	Avatar_tmpl     string
	Agreed_to_terms bool
	Registered      time.Time
	Customer_id   string
	password_key  string
	password_salt string
	talk          *talk.User
	customer      *stripe.Customer
	*Admin
	*Student
	*Members
}

// New members are created with an uninitialized password, which must be set via
//	the reset form.
func (ms *Members) New_member(username, name, email string) (*Member, error) {
	m := &Member{
		Username: username,
		Name:     name,
		Email:    email,
		Members:  ms}
	if err := ms.Validate_username(username); err != nil {
		return nil, err
	}
	if err := Validate_name(name); err != nil {
		return nil, err
	}
	if err := Validate_email(email); err != nil {
		return nil, err
	}
	if !ms.Username_available(username) {
		return nil, fmt.Errorf("Username is already in use")
	}
	if !ms.Email_available(email) {
		return nil, fmt.Errorf("E-mail address is already in use")
	}
	if err := m.QueryRow(
		"INSERT INTO member"+
			"	(username, name, email) "+
			"VALUES ($1, $2, $3) "+
			"RETURNING id, registered",
		username, name, email).Scan(&m.Id, &m.Registered);
		err != nil {
		log.Panic(err)
	}
	if err := m.sync_talk_user(); err != nil {
		m.Delete_member()
		return nil, err
	}
	return m, nil
}

func (m *Member) sync_talk_user() error {
	talk, err := m.Talk.Sync(m.Id, m.Username, m.Email, m.Name)
	if err != nil {
		return err
	}
	m.talk = talk
	m.Update_avatar_tmpl(talk.Avatar_tmpl)
	return nil
}

//TODO: cascade through all tables
func (m *Member) Delete_member() {
	if _, err := m.Exec("DELETE FROM member WHERE id = $1", m.Id); err != nil {
		log.Panic(err)
	}
}

func (m *Member) Set_password(password string) {
	m.password_salt = Rand256()
	m.password_key = key(password, m.password_salt)
	if _, err := m.Exec(
		"UPDATE member "+
		"SET password_key = $1,"+
		"	password_salt = $2 "+
		"WHERE id = $3",
		m.password_key, m.password_salt, m.Id); err != nil {
		log.Panic(err)
	}
	if _, err := m.Exec(
		"DELETE FROM reset_password_token "+
		"WHERE member = $1", m.Id); err != nil {
		log.Panic(err)
	}
}

func (m *Member) Update_username(username string) error {
	if username == m.Username {
		return nil
	}
	if err := m.Validate_username(username); err != nil {
		return err
	}
	if !m.Username_available(username) {
		return fmt.Errorf("Username is already in use")
	}
	m.Username = username
	if _, err := m.Exec("UPDATE member "+
		"SET username = $1 "+
		"WHERE id = $2", username, m.Id); err != nil {
		log.Panic(err)
	}
	return m.sync_talk_user()
}

func (m *Member) Update_name(name string) error {
	if name == m.Name {
		return nil
	}
	if err := Validate_name(name); err != nil {
		return err
	}
	m.Name = name
	if _, err := m.Exec("UPDATE member "+
		"SET name = $1 "+
		"WHERE id = $2", name, m.Id); err != nil {
		log.Panic(err)
	}
	return m.sync_talk_user()
}

func (m *Member) Update_email(email string) error {
	if email == m.Email {
		return nil
	}
	if err := Validate_email(email); err != nil {
		return err
	}
	if !m.Email_available(email) {
		return fmt.Errorf("E-mail address is already in use")
	}
	if t, _ := m.Talk.Get_user_by_email(email); t != nil {
		log.Printf("Talk user re-association attempt failed: @%s <%s> -> <%s>",
			m.Username, m.Email, email)
		return fmt.Errorf("E-mail address is already in use by a Talk user, "+
			"cannot re-associate account")
	}
	m.Email = email
	m.Delete_verification_tokens(email)
	if _, err := m.Exec("UPDATE member "+
		"SET email = $1 "+
		"WHERE id = $2", email, m.Id); err != nil {
		log.Panic(err)
	}
	return m.sync_talk_user()
}

func (m *Member) Set_registration_date(date time.Time) {
	if date.IsZero() {
		date = time.Now()
	}
	m.Registered = date
	if _, err := m.Exec("UPDATE member "+
		"SET registered = $1 "+
		"WHERE id = $2", date, m.Id); err != nil {
		log.Panic(err)
	}
}

var Key_card_rexp = regexp.MustCompile(`^[0-9]{2}:[0-9]{5}$`)

func (m *Member) Set_key_card(key_card string) error {
	if !Key_card_rexp.MatchString(key_card) {
		return fmt.Errorf("Invalid key-card format")
	}
	var n int
	if err := m.QueryRow(
		"SELECT COUNT(*) "+
			"FROM member "+
			"WHERE key_card = $1",
		key_card).Scan(&n); err != nil {
		log.Panic(err)
	}
	if n != 0 {
		return fmt.Errorf("Key-card already in use")
	}
	m.Key_card = key_card
	if _, err := m.Exec("UPDATE member "+
		"SET key_card = $1 "+
		"WHERE id = $2", key_card, m.Id); err != nil {
		log.Panic(err)
	}
	return nil
}

//TODO: validate input
func (m *Member) Set_telephone(tel string) error {
	m.Telephone = tel
	if _, err := m.Exec("UPDATE member "+
		"SET telephone = $1 "+
		"WHERE id = $2", tel, m.Id); err != nil {
		log.Panic(err)
	}
	return nil
}

func (m *Member) Update_avatar_tmpl(avatar_tmpl string) {
	if m.Avatar_tmpl == avatar_tmpl {
		return
	}
	m.Avatar_tmpl = avatar_tmpl
	if _, err := m.Exec("UPDATE member "+
		"SET avatar_tmpl = $1 "+
		"WHERE id = $2", avatar_tmpl, m.Id); err != nil {
		log.Panic(err)
	}
}

func (m *Member) Authenticate(password string) bool {
	if m.password_key == key(password, m.password_salt) {
		return true
	}
	return false
}

func (m *Member) Avatar_url(size int) string {
	return strings.Replace(m.Avatar_tmpl, "{size}", fmt.Sprint(size), 1)
}

func (m *Member) create_reset_token() string {
	token := Rand256()
	if _, err := m.Exec("INSERT INTO reset_password_token (member, token) "+
		"VALUES ($1, $2) "+
		"ON CONFLICT (member) DO UPDATE SET"+
		"	(token, time) = ($2, now())", m.Id, token); err != nil {
		log.Panic("Failed to set password reset token: ", err)
	}
	return token
}

func (m *Member) Send_password_reset(domain string) {
	token := m.create_reset_token()
	if token == "" {
		return
	}
	msg := message{subject: "Makerspace.ca: password reset"}
	msg.set_from("Makerspace", "admin@makerspace.ca")
	msg.add_to(m.Name, m.Email)
	//See TODO in /member/admin.go: Force_password_reset()
	msg.body = "Hello " + m.Name + " (@" + m.Username + "),\n\n" +
		"A password reset has been requested for your account.  " +
		"If you did not initiate this request, please ignore this e-mail.\n\n" +
		"Reset your makerspace password by visiting " +
		domain + "/sso/reset?token=" + token + ".\n\n" +
		"Your password-reset token will expire in " +
		m.Config.Password_reset_window + ", you can request a new" +
		"token at " + domain + "/sso/reset?username=" +
		url.QueryEscape(m.Username) + "&email=" + url.QueryEscape(m.Email) +
		".\n\n"
	m.send_email("admin@makerspace.ca", msg.emails(), m.format_message(msg))
}

func (ms *Members) Send_email_verification(email, body string, m *Member) {
	token := Rand256()
	var err error
	if m == nil {
		_, err = ms.Exec(
			"INSERT INTO email_verification_token"+
			"	(token, email) "+
			"VALUES ($1, $2)", token, email)
	} else {
		_, err = ms.Exec(
			"INSERT INTO email_verification_token"+
			"	(token, email, member) "+
			"VALUES ($1, $2, $3) "+
			"ON CONFLICT (member) DO UPDATE "+
			"SET (token, email, time) = ($1, $2, now())", token, email, m.Id)
	}
	if err != nil {
		log.Panic("Failed to set email verification token: ", err)
	}
	msg := message{
		subject: "E-mail verification",
		body: body + token}
	msg.set_from("Makerspace", "admin@makerspace.ca")
	if m != nil {
		msg.add_to(m.Name, email)
	} else {
		msg.add_to("", email)
	}
	ms.send_email("admin@makerspace.ca", msg.emails(), ms.format_message(msg))
}

func (m *Member) Get_pending_subscriptions() []*Pending_subscription {
	pending := make([]*Pending_subscription, 0)
	rows, err := m.Query(
		"SELECT requested_at, plan_id " +
		"FROM pending_subscription " +
		"WHERE member = $1 " +
		"ORDER BY requested_at DESC", m.Id)
	defer rows.Close()
	if err != nil && err != sql.ErrNoRows {
		log.Panic(err)
	}
	for rows.Next() {
		p := Pending_subscription{Member: m}
		if err = rows.Scan(&p.Requested_at, &p.Plan_id); err != nil {
			log.Panic(err)
		}
		pending = append(pending, &p)
	}
	return pending
}

func (m *Member) Talk_user() *talk.User {
	if m.talk == nil {
		m.talk, _ = m.Talk.Get_user_by_external_id(m.Id)
		if m.talk != nil && m.Avatar_tmpl != m.talk.Avatar_tmpl {
			m.Update_avatar_tmpl(m.talk.Avatar_tmpl)
		}
	}
	return m.talk
}

// Last_seen returns the last page-load time in a session by member <m>.
//	ls.IsZero() == true if <m> has never created a session.
func (m *Member) Last_seen() time.Time {
	var ls pq.NullTime
	if err := m.QueryRow(
		"SELECT max(last_seen) "+
			"FROM session_http "+
			"WHERE member = $1", m.Id).Scan(&ls); err != nil && err != sql.ErrNoRows {
		log.Panic(err)
	}
	return ls.Time
}
