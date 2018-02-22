package app

import (
	"database/sql"
	_ "github.com/lib/pq"
)

type Application struct {
	conf *Config
	db *sql.DB
}

// New allocates and starts a new application instance
func New(conf Config) (*Application, error) {
	app := &Application{conf: &conf}
	db, err := sql.Open("postgres", conf.Database)
	if err != nil {
		return nil, err
	}
	app.db = db
	return app, nil
}

func Reload(conf Config) error {
	return nil
}

func Stop() error {
	return nil
}
