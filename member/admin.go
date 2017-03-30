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

func (ms *Members) List_pending_approvals() []*Member {
	members := make([]*Member, 0)
	rows, err := ms.Query(
		"SELECT i.member "+
		"FROM invoice i "+
		"JOIN fee f "+
		"ON i.fee = f.id "+
		"WHERE f.category = 'membership'"+
		"	AND i.start_date IS NULL"+
		"	AND (i.end_date < now() OR i.end_date IS NULL)")
	defer rows.Close()
	if err != nil {
		if err == sql.ErrNoRows {
			return members
		}
		log.Panic(err)
	}
	for rows.Next() {
		var member_id int
		if err = rows.Scan(&member_id); err != nil {
			log.Panic(err)
		}
		members = append(members, ms.Get_member_by_id(member_id))
	}
	return members
}
