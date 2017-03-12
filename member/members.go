package member

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"github.com/vvanpo/makerspace/billing"
	"github.com/vvanpo/makerspace/talk"
	"golang.org/x/crypto/scrypt"
	"log"
	"regexp"
)

type Members struct {
	Config map[string]interface{}
	*sql.DB
	*talk.Talk_api
	*billing.Billing
}

func Rand256() string {
	n := make([]byte, 32)
	_, err := rand.Read(n)
	if err != nil {
		log.Panic(err)
	}
	return hex.EncodeToString(n)
}

func key(password, salt string) string {
	s, err := hex.DecodeString(salt)
	if err != nil {
		log.Panicf("Invalid salt: %s", err)
	}
	key, err := scrypt.Key([]byte(password), s, 16384, 8, 1, 32)
	if err != nil {
		log.Panic(err)
	}
	return hex.EncodeToString(key)
}

var username_chars_rexp = regexp.MustCompile(`[^\w.-]`)
var username_first_char_rexp = regexp.MustCompile(`^[\W]`)
var username_last_char_rexp = regexp.MustCompile(`[^A-Za-z0-9]$`)
var username_double_special_rexp = regexp.MustCompile(`[-_.]{2,}`)
var username_confusing_suffix_rexp = regexp.MustCompile(`\.(js|json|css|htm|html|xml|jpg|jpeg|png|gif|bmp|ico|tif|tiff|woff)$`)
func (ms *Members) Check_username_availability(username string) (available bool, err string) {
	if username == "" {
		return false, "Username cannot be blank"
	} else if len(username) < 3 {
		return false, "Username must be at least 3 characters"
	} else if len(username) > 20 {
		return false, "Username must be no more than 20 characters"
	} else if username_chars_rexp.MatchString(username) {
		return false, "Username must only include numbers, letters, underscores, hyphens, and periods"
	} else if username_first_char_rexp.MatchString(username) {
		return false, "Username must begin with an underscore or alphanumeric character"
	} else if username_last_char_rexp.MatchString(username) {
		return false, "Username must end with an alphanumeric character"
	} else if username_double_special_rexp.MatchString(username) {
		return false, "Username cannot contain consecutive special characters (underscore, period, or hyphen)"
	} else if username_confusing_suffix_rexp.MatchString(username) {
		return false, "Username must not end in a confusing filetype suffix"
	}
	for _, u := range ms.Config["reserved_usernames"].([]interface{}) {
		if username == u.(string) {
			return false, "Username reserved"
		}
	}
	var count int
	if err := ms.QueryRow(
		"SELECT COUNT(*) "+
			"FROM member "+
			"WHERE username = $1", username).Scan(&count); err != nil {
		log.Panic(err)
	}
	if count == 1 {
		return false, "Username already in use"
	}
	return ms.Check_username(username)
}

var email_rexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&’*+/=?^_`{|}~-]+@[a-zA-Z0-9-]+(?:\\.[a-zA-Z0-9-]+)*$")
func (ms *Members) Check_email_availability(email string) (available bool, err string) {
	if email == "" {
		return false, "E-mail cannot be blank"
	}
	if !email_rexp.MatchString(email) {
		return false, "Invalid E-mail address"
	}
	var count int
	if err := ms.QueryRow(
		"SELECT COUNT(*) "+
			"FROM member "+
			"WHERE email = $1", email).Scan(&count); err != nil {
		log.Panic(err)
	}
	if count == 0 {
		return true, ""
	}
	return false, "E-mail already in use"
}

var name_rexp = regexp.MustCompile(`^([\pL\pN\pM\pP]+ ?)+$`)

// New creates a new user, returns nil and a set of errors on invalid input.
func (ms *Members) New_member(username, name, email, password string) (m *Member, err map[string]string) {
	err = make(map[string]string)
	salt := Rand256()
	m = &Member{
		Username:      username,
		Name:          name,
		Email:         email,
		password_key:  key(password, salt),
		password_salt: salt,
		Members:       ms}
	if !name_rexp.MatchString(name) {
		err["name_error"] = "Invalid characters in name"
		m = nil
	} else if len(name) > 100 {
		err["name_error"] = "Name must be no more than 100 characters"
		m = nil
	}
	if available, e := ms.Check_username_availability(username); !available {
		err["username_error"] = e
		m = nil
	}
	if available, e := ms.Check_email_availability(email); !available {
		err["email_error"] = e
		m = nil
	}
	if m == nil {
		return
	}
	if e := m.QueryRow(
		"INSERT INTO member ("+
			"	username,"+
			"	name,"+
			"	password_key,"+
			"	password_salt,"+
			"	email"+
			") "+
			"VALUES ($1, $2, $3, $4, $5) "+
			"RETURNING id, registered",
		username, name, m.password_key, salt, email).Scan(&m.Id, &m.Registered);
		e != nil {
		log.Panic(e)
	}
	m.talk = ms.Sync(m.Id, m.Username, m.Email, m.Name)
	if m.talk == nil {
		m.Delete_member()
		return nil, err
	}
	return m, nil
}

func (ms *Members) Get_all_members() []*Member {
	members := make([]*Member, 0)
	rows, err := ms.Query(
		"SELECT " +
			"	m.id, " +
			"	m.username, " +
			"	m.name, " +
			"	m.password_key, " +
			"	m.password_salt, " +
			"	m.email, " +
			"	m.agreed_to_terms, " +
			"	m.registered, " +
			"	s.username IS NOT NULL, " +
			"	a.username IS NOT NULL " +
			"FROM member m " +
			"NATURAL LEFT JOIN administrator a " +
			"NATURAL LEFT JOIN student s")
	defer rows.Close()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return members
	}
	for rows.Next() {
		var password_key, password_salt sql.NullString
		m := &Member{Members: ms}
		if err = rows.Scan(&m.Id, &m.Username, &m.Name, &password_key,
			&password_salt, &m.Email, &m.Agreed_to_terms, &m.Registered,
			&m.Student, &m.Admin); err != nil {
			log.Panic(err)
		}
		m.password_key = password_key.String
		m.password_salt = password_salt.String
		members = append(members, m)
	}
	return members
}

/*
func (ms *Members) Get_all_active_members() []*Member {
	members := make([]*Member, 0)
	//TODO: BUG: should by on f.category = 'membership'
	rows, err := ms.db.Query("SELECT m.id, m.username, m.name, m.password_key, m.password_salt, m.email, m.agreed_to_terms, m.registered, s.username IS NOT NULL, a.username IS NOT NULL FROM member m NATURAL LEFT JOIN administrator a NATURAL LEFT JOIN student s JOIN (SELECT COALESCE(i.paid_by, i.username) AS paid_by FROM invoice i LEFT JOIN fee f ON (i.fee = f.id) WHERE COALESCE(i.recurring, f.recurring) = '1 month' AND (i.end_date > now() OR i.end_date IS NULL)) inv ON inv.paid_by = m.username")
	defer rows.Close()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return members
	}
	for rows.Next() {
		m := &Member{Members: ms}
		if err = rows.Scan(&m.Id, &m.Username, &m.Name, &m.password_key, &m.password_salt, &m.Email, &m.Agreed_to_terms, &m.Registered, &m.Student, &m.Admin); err != nil {
			log.Panic(err)
		}
	}
	return members
}*/
