package member

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/stripe/stripe-go"
	"github.com/vvanpo/makerspace/talk"
	"golang.org/x/crypto/scrypt"
	"log"
	"regexp"
	"time"
)

type Config struct {
	Reserved_usernames        []string
	Password_reset_window     string
	Smtp                      struct {
		Address  string
		Port     int
		Username string
		Password string
	}
	Billing struct {
		Private_key string
		Public_key  string
	}
}

type Members struct {
	Config
	*sql.DB
	*talk.Talk_api
	Plans map[string]*stripe.Plan
}

func New(config Config, db *sql.DB, talk *talk.Talk_api) *Members {
	stripe.Key = config.Billing.Private_key
	stripe.LogLevel = 1
	ms := &Members{config, db, talk, make(map[string]*stripe.Plan)}
	ms.load_plans()
	return ms
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

func (ms *Members) Validate_username(username string) error {
	var err_string string
	if username == "" {
		err_string = "Username cannot be blank"
	} else if len(username) < 3 {
		err_string = "Username must be at least 3 characters"
	} else if len(username) > 20 {
		err_string = "Username must be no more than 20 characters"
	} else if username_chars_rexp.MatchString(username) {
		err_string = "Username must only include numbers, letters, "+
			"underscores, hyphens, and periods"
	} else if username_first_char_rexp.MatchString(username) {
		err_string = "Username must begin with an underscore or alphanumeric "+
			"character"
	} else if username_last_char_rexp.MatchString(username) {
		err_string = "Username must end with an alphanumeric character"
	} else if username_double_special_rexp.MatchString(username) {
		err_string = "Username cannot contain consecutive special characters "+
			"(underscore, period, or hyphen)"
	} else if username_confusing_suffix_rexp.MatchString(username) {
		err_string = "Username must not end in a confusing filetype suffix"
	}
	for _, u := range ms.Config.Reserved_usernames {
		if username == u {
			err_string = "Username reserved"
		}
	}
	if err_string != "" {
		return fmt.Errorf(err_string)
	}
	return nil
}

var name_rexp = regexp.MustCompile(`^([\pL\pN\pM\pP]+ ?)+$`)

func Validate_name(name string) error {
	if name == "" {
		return fmt.Errorf("Name cannot be blank")
	} else if len(name) > 100 {
		return fmt.Errorf("Name must be no more than 100 characters")
	} else if !name_rexp.MatchString(name) {
		return fmt.Errorf("Name contains invalid characters")
	}
	return nil
}

var email_rexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&’*+/=?^_`{|}~-]+@[a-zA-Z0-9-]+(?:\\.[a-zA-Z0-9-]+)*$")

func Validate_email(email string) error {
	if email == "" {
		return fmt.Errorf("E-mail address cannot be blank")
	} else if !email_rexp.MatchString(email) {
		return fmt.Errorf("Invalid e-mail address format")
	}
	return nil
}

func (ms *Members) Username_available(username string) bool {
	var count int
	if err := ms.QueryRow(
		"SELECT COUNT(*) "+
			"FROM member "+
			"WHERE username = $1", username).Scan(&count); err != nil {
		log.Panic(err)
	}
	if count == 0 {
		return true
	}
	return false
}

func (ms *Members) Email_available(email string) bool {
	var count int
	if err := ms.QueryRow(
		"SELECT COUNT(*) "+
			"FROM member "+
			"WHERE email = $1", email).Scan(&count); err != nil {
		log.Panic(err)
	}
	if count == 0 {
		return true
	}
	return false
}

func parse_duration(w string) (time.Duration, error) {
	var weeks int
	if w == "1 week" {
		w = fmt.Sprintf("%dh", 7*24)
	} else if n, err := fmt.Sscanf(w, "%d weeks", &weeks); n == 1 && err == nil {
		w = fmt.Sprintf("%dh", 7*24*weeks)
	}
	return time.ParseDuration(w)
}

func (ms *Members) Get_member_from_reset_token(token string) (*Member, error) {
	var member_id int
	var t time.Time
	if err := ms.QueryRow(
		"SELECT member, time "+
			"FROM reset_password_token "+
			"WHERE token = $1", token).Scan(&member_id, &t); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("Reset token does not exist")
		}
		log.Panic(err)
	}
	duration, err := parse_duration(ms.Config.Password_reset_window)
	if err != nil {
		log.Panic(err)
	}
	expires := t.Add(duration)
	if time.Now().After(expires) {
		if _, err := ms.Exec("DELETE FROM reset_password_token "+
			"WHERE token = $1", token); err != nil {
			log.Panic(err)
		}
		return nil, fmt.Errorf("Reset token is expired")
	}
	return ms.Get_member_by_id(member_id), nil
}

func (ms *Members) Verify_email_token(token string) (email string, m *Member) {
	var (
		member_id sql.NullInt64
	)
	if err := ms.QueryRow(
		"SELECT member, email "+
			"FROM email_verification_token "+
			"WHERE token = $1", token).Scan(&member_id, &email); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		log.Panic(err)
	}
	if member_id.Valid {
		return email, ms.Get_member_by_id(int(member_id.Int64))
	}
	return email, nil
}

func (ms *Members) Delete_verification_tokens(email string) {
	if _, err := ms.Exec(
		"DELETE FROM email_verification_token "+
		"WHERE email = $1", email); err != nil {
		log.Panic(err)
	}
}
