package member

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/vvanpo/makerspace/billing"
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
	Avatar_tmpl     string
	Telephone       string
	Agreed_to_terms bool
	Registered      time.Time
	Gratuitous      bool
	Approved        bool
	*Admin
	*Student
	*Members
	Membership_invoice *billing.Invoice
	password_key       string
	password_salt      string
	talk               *talk.Talk_user
	payment            *billing.Profile
}

//TODO: check corporate account
func (ms *Members) Get_member_by_id(id int) *Member {
	m := &Member{Id: id, Members: ms}
	var (
		email, key_card, password_key, password_salt, avatar_tmpl,
		telephone sql.NullString
		approved_at pq.NullTime
	)
	if err := m.QueryRow(
		"SELECT"+
			"	username,"+
			"	name,"+
			"	password_key,"+
			"	password_salt,"+
			"	email,"+
			"	key_card,"+
			"	avatar_tmpl,"+
			"	telephone,"+
			"	agreed_to_terms,"+
			"	registered,"+
			"	gratuitous,"+
			"	approved_at "+
			"FROM member "+
			"WHERE id = $1",
		id).Scan(&m.Username, &m.Name, &password_key, &password_salt, &email,
		&key_card, &avatar_tmpl, &telephone, &m.Agreed_to_terms, &m.Registered,
		&m.Gratuitous, &approved_at); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	m.password_key = password_key.String
	m.password_salt = password_salt.String
	m.Email = email.String
	m.Key_card = key_card.String
	m.Avatar_tmpl = avatar_tmpl.String
	m.Telephone = telephone.String
	if approved_at.Valid {
		m.Approved = true
	}
	m.get_student()
	m.get_admin()
	if m.Payment() != nil {
		m.Membership_invoice = m.payment.Get_membership()
	}
	return m
}

func (ms *Members) Get_member_by_username(username string) *Member {
	var member_id int
	if err := ms.QueryRow(
		"SELECT id "+
			"FROM member "+
			"WHERE username = $1",
		username).Scan(&member_id); err != nil {
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

func (m *Member) Set_name(name string) error {
	if _, err := validate_name(name); err != nil {
		return err
	}
	m.Name = name
	if _, err := m.Exec("UPDATE member "+
		"SET name = $1 "+
		"WHERE id = $2", name, m.Id); err != nil {
		log.Panic(err)
	}
	return nil
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

func (m *Member) set_email(email string) {
	m.Email = email
	if _, err := m.Exec("UPDATE member "+
		"SET email = $1 "+
		"WHERE id = $2", email, m.Id); err != nil {
		log.Panic(err)
	}
}

func (m *Member) set_gratuitous(free bool) {
	m.Gratuitous = free
	if _, err := m.Exec("UPDATE member "+
		"SET gratuitous = $1 "+
		"WHERE id = $2", free, m.Id); err != nil {
		log.Panic(err)
	}
}

func (m *Member) set_avatar_tmpl(avatar_tmpl string) {
	m.Avatar_tmpl = avatar_tmpl
	if _, err := m.Exec("UPDATE member "+
		"SET avatar_tmpl = $1 "+
		"WHERE id = $2", avatar_tmpl, m.Id); err != nil {
		log.Panic(err)
	}
}

func (m *Member) Avatar_url(size int) string {
	return strings.Replace(m.Avatar_tmpl, "{size}", fmt.Sprint(size), 1)
}

func (m *Member) create_reset_token() string {
	if !m.Verified_email() {
		return ""
	}
	token := Rand256()
	if _, err := m.Exec("INSERT INTO reset_password_token (member, token) "+
		"VALUES ($1, $2) "+
		"ON CONFLICT (member) DO UPDATE SET"+
		"	(token, time) = ($2, now())", m.Id, token); err != nil {
		log.Panic("Failed to set password reset token: ", err)
	}
	return token
}

func (m *Member) Send_password_reset() {
	token := m.create_reset_token()
	if token == "" {
		return
	}
	msg := message{subject: "Makerspace.ca: password reset"}
	msg.set_from("Makerspace", "admin@makerspace.ca")
	msg.add_to(m.Name, m.Email)
	//TODO use config.json value for domain
	msg.body = "Hello " + m.Name + " (@" + m.Username + "),\n\n" +
		"A password reset has been requested for your account.  " +
		"If you did not initiate this request, please ignore this e-mail.\n\n" +
		"Reset your makerspace password by visiting " +
		m.Config["url"].(string) + "/sso/reset?token=" + token + ".\n\n" +
		"Your password-reset token will expire in " +
		m.Config["password-reset-window"].(string) + ", you can request a new" +
		"token at " + m.Config["url"].(string) + "/sso/reset?username=" +
		url.QueryEscape(m.Username) + "&email=" + url.QueryEscape(m.Email) +
		".\n\n"
	m.send_email("admin@makerspace.ca", msg.emails(), msg.format())
}

func (m *Member) Send_email_verification(email string) {
	token := Rand256()
	if _, err := m.Exec(
		"INSERT INTO email_verification_token"+
			"	(member, email, token) "+
			"VALUES ($1, $2, $3) "+
			"ON CONFLICT (member) DO UPDATE SET"+
			"	(email, token, time) = ($2, $3, now())", m.Id, email, token); err != nil {
		log.Panic("Failed to set email verification token: ", err)
	}
	msg := message{subject: "Makerspace.ca: e-mail verification"}
	msg.set_from("Makerspace", "admin@makerspace.ca")
	msg.add_to(m.Name, email)
	msg.body = "Hello " + m.Name + " (@" + m.Username + "),\n\n" +
		"To sign-in to your Makerspace account, you must first verify that " +
		"are the owner of this associated e-mail address.\n\n" +
		"If the above name and username is correct, please verify your " +
		"e-mail address (" + email + ") by visiting " +
		m.Config["url"].(string) + "/sso/verify-email?token=" + token + "\n\n" +
		"Your verification token will expire in " +
		m.Config["email-verification-window"].(string) + ", you can request " +
		"a new token at " + m.Config["url"].(string) +
		"/sso/verify-email?username=" + url.QueryEscape(m.Username) +
		"&email=" + url.QueryEscape(email) + ".\n\n"
	m.send_email("admin@makerspace.ca", msg.emails(), msg.format())
}

func (m *Member) Verify_email(email string) error {
	m.talk = m.Sync(m.Id, m.Username, email, m.Name)
	if m.talk == nil {
		return fmt.Errorf("Failed to sync talk user: (%d) %s <%s>\n", m.Id,
			m.Username, email)
	} else {
		m.set_avatar_tmpl(m.talk.Avatar_tmpl)
	}
	m.set_email(email)
	//TODO: delete unverified members with this pending verification
	if _, err := m.Exec(
		"DELETE FROM email_verification_token "+
			"WHERE email = $1 "+
			"	OR member = $2", email, m.Id); err != nil {
		log.Panic(err)
	}
	return nil
}

func (m *Member) Talk_user() *talk.Talk_user {
	if !m.Verified_email() {
		return nil
	} else if m.talk == nil {
		m.talk = m.Talk_api.Get_user(m.Id)
		if m.talk != nil && m.Avatar_tmpl != m.talk.Avatar_tmpl {
			m.set_avatar_tmpl(m.talk.Avatar_tmpl)
		}
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
	//TODO: propagate errors
	if m.Membership_invoice != nil && m.Approved {
		m.payment.Approve_pending_membership(m.Membership_invoice)
		m.set_gratuitous(false)
	}
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
	if m.Talk_user() != nil {
		m.Talk_user().Remove_from_group("Members")
	}
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
