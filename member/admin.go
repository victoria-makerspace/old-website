package member

import (
	"database/sql"
	"log"
)

type Admin struct {
	privileges []string
}

func (m *Member) get_admin() {
	admin := &Admin{}
	if err := m.QueryRow(
		"SELECT privileges "+
		"FROM administrator "+
		"WHERE member = $1", m.Id).
		Scan(&admin.privileges); err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return
	}
	m.Admin = admin
}
