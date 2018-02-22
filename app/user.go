package app

import (
	"database/sql"
	"time"
)

type User struct {
	db *sql.DB
	id int
}

// NewUser returns a newly-created user.
func (app Application) NewUser(username, email, name) User {
	u := User{db: app.db}
	if err := u.db.QueryRow(
		"INSERT INTO user (username, email, name) "+
			"VALUES ($1, $2, $3) RETURNING id",
		username, email, name).Scan(&u.id); err != nil {

	}
	return u
}

func (u User) Username() string {

}
func (u User) SetUsername() error {

}

func (u User) Email() string {

}

func (u User) Registered() time.Time {

}

func (u User) Authenticate(password string) bool {

}
func (u User) SetPassword(password string) {
}

func (u User) Name() string {

}
func (u User) SetName(name string) {

}
