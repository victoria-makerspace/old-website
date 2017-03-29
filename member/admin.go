package member

import (
	"database/sql"
	"github.com/lib/pq"
	"log"
)

type Admin struct {
	privileges []string
}

func (m *Member) get_admin() {
	var privileges pq.StringArray
	if err := m.QueryRow(
		"SELECT privileges "+
			"FROM administrator "+
			"WHERE member = $1", m.Id).
		Scan(&privileges); err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return
	}
	m.Admin = &Admin{privileges}
}

func (a *Member) Approve_member(m *Member) {
	if a.Admin == nil {
		log.Panicf("%s is not an administrator\n", a.Username)
	}
	if _, err := m.Exec(
		"UPDATE member "+
		"SET"+
		"	approved_at = now(),"+
		"	approved_by = $1 ", a.Id); err != nil {
		log.Panic(err);
	}
	m.Approved = true
}
