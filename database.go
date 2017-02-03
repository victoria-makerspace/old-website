package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"log"
)

func Database(conninfo map[string]string) *sql.DB {
	var conn string
	for k, v := range conninfo {
		conn += " " + k + "=" + v
	}
	db, err := sql.Open("postgres", conn)
	if err != nil {
		log.Fatal(err)
	}
	return db
}
