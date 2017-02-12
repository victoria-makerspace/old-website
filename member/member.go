package member

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"golang.org/x/crypto/scrypt"
	"log"
	"time"
)

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

type Member struct {
	Username      string
	Name          string
	Email         string
	Registered    time.Time
	Active        bool
	Student	      bool
	password_key  string
	password_salt string
	db            *sql.DB
}

// New creates a new user, but will panic if the username already exists
func New(username, name, email, password string, db *sql.DB) *Member {
	salt := Rand256()
	m := &Member{
		Username:      username,
		Name:          name,
		Email:         email,
		Registered:    time.Now(),
		password_key:  key(password, salt),
		password_salt: salt,
		db:            db}
	_, err := db.Exec("INSERT INTO member (username, name, password_key, password_salt, email, registered) VALUES ($1, $2, $3, $4, $5, $6)", username, name, m.password_key, salt, email, m.Registered)
	if err != nil {
		log.Panic(err)
	}
	return m
}

func Get(username string, db *sql.DB) *Member {
	m := &Member{db: db}
	// Populate m and check if member is active, by asserting whether or not
	//	they are being currently invoiced.
	if err := db.QueryRow("SELECT username, name, password_key, password_salt, email, registered, EXISTS (SELECT 1 FROM invoice i INNER JOIN fee f ON (i.fee = f.id) WHERE i.username = $1 AND f.category = 'membership' AND (i.end_date > now() OR i.end_date IS NULL)), EXISTS (SELECT 1 FROM student WHERE username = $1) FROM member WHERE username = $1", username).Scan(&m.Username, &m.Name, &m.password_key, &m.password_salt, &m.Email, &m.Registered, &m.Active, &m.Student); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	return m
}

func (m *Member) Authenticate(password string) bool {
	if m.password_key == key(password, m.password_salt) {
		return true
	}
	return false
}

