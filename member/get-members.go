package member

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"log"
	"sort"
)

func (ms *Members) get_members_by_query(where_cond string, values ...interface{}) map[int]*Member {
	members := make(map[int]*Member)
	rows, err := ms.Query(
		"SELECT"+
			"	m.id,"+
			"	m.username,"+
			"	m.name,"+
			"	m.email,"+
			"	m.key_card,"+
			"	m.telephone,"+
			"	m.avatar_tmpl,"+
			"	m.agreed_to_terms,"+
			"	m.registered,"+
			"	m.stripe_customer_id,"+
			"	m.password_key,"+
			"	m.password_salt,"+
			"	a.member,"+
			"	a.privileges,"+
			"	s.member,"+
			"	s.institution,"+
			"	s.student_email,"+
			"	s.graduation_date "+
			"FROM member m "+
			"LEFT JOIN administrator a "+
			"ON m.id = a.member "+
			"LEFT JOIN student s "+
			"ON m.id = s.member "+
			where_cond, values...)
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			m                                             Member
			key_card, telephone, avatar_tmpl, customer_id sql.NullString
			password_key, password_salt                   sql.NullString
			admin, student                                sql.NullInt64
			privileges                                    pq.StringArray
			institution, student_email                    sql.NullString
			graduation_date                               pq.NullTime
		)
		if err := rows.Scan(&m.Id, &m.Username, &m.Name, &m.Email, &key_card,
			&telephone, &avatar_tmpl, &m.Agreed_to_terms, &m.Registered,
			&customer_id, &password_key, &password_salt,
			&admin, &privileges,
			&student, &institution, &student_email, &graduation_date); err != nil {
			log.Panic(err)
		}
		m.Key_card = key_card.String
		m.Telephone = telephone.String
		m.Avatar_tmpl = avatar_tmpl.String
		m.Customer_id = customer_id.String
		m.password_key = password_key.String
		m.password_salt = password_salt.String
		m.Members = ms
		if admin.Valid {
			m.Admin = &Admin{privileges}
		}
		if student.Valid {
			m.Student = &Student{
				institution.String,
				student_email.String,
				graduation_date.Time}
		}
		members[m.Id] = &m
	}
	return members
}

func (ms *Members) Get_member_by_id(id int) *Member {
	query := "WHERE m.id = $1"
	for _, m := range ms.get_members_by_query(query, id) {
		return m
	}
	return nil
}

func (ms *Members) Get_member_by_username(username string) *Member {
	query := "WHERE m.username = $1"
	for _, m := range ms.get_members_by_query(query, username) {
		return m
	}
	return nil
}

//TODO: canonicalize e-mail address
func (ms *Members) Get_member_by_email(email string) *Member {
	query := "WHERE m.email = $1"
	for _, m := range ms.get_members_by_query(query, email) {
		return m
	}
	return nil
}

func (ms *Members) Get_member_by_customer_id(customer_id string) *Member {
	query := "WHERE m.stripe_customer_id = $1"
	for _, m := range ms.get_members_by_query(query, customer_id) {
		return m
	}
	return nil
}

func (ms *Members) Get_members_by_name(name string) map[int]*Member {
	query := "WHERE m.name = $1"
	return ms.get_members_by_query(query, name)
}

type member_list struct {
	list []*Member
	less func(i, j int) bool
}

// Methods to implement sort.Interface
func (m member_list) Len() int {
	return len(m.list)
}
func (m member_list) Swap(i, j int) {
	m.list[i], m.list[j] = m.list[j], m.list[i]
}
func (m member_list) Less(i, j int) bool {
	return m.less(i, j)
}

func (ms *Members) list_members_by_query(less func(m []*Member) func(i, j int) bool, cond string, values ...interface{}) []*Member {
	members := ms.get_members_by_query(cond, values...)
	ml := member_list{list: make([]*Member, 0, len(members))}
	for _, m := range members {
		ml.list = append(ml.list, m)
	}
	ml.less = less(ml.list)
	sort.Sort(ml)
	return ml.list
}

// Ordered by username
func (ms *Members) List_members() []*Member {
	less := func(m []*Member) func(i, j int) bool {
		return func(i, j int) bool {
			return m[i].Username < m[j].Username
		}
	}
	return ms.list_members_by_query(less, "")
}

// Ordered by last-seen time
func (ms *Members) List_active_members() []*Member {
	less := func(m []*Member) func(i, j int) bool {
		return func(i, j int) bool {
			return m[i].Last_seen().Unix() > m[j].Last_seen().Unix()
		}
	}
	query := "JOIN session_http sh " +
		"ON m.id = sh.member " +
		"GROUP BY m.id, a.member, s.member " +
		"ORDER BY max(sh.last_seen)"
	return ms.list_members_by_query(less, query)
}

// Ordered by registration date
func (ms *Members) List_new_members(limit int) []*Member {
	less := func(m []*Member) func(i, j int) bool {
		return func(i, j int) bool {
			return m[i].Registered.Unix() > m[j].Registered.Unix()
		}
	}
	query := "ORDER BY registered DESC " +
		"LIMIT " + fmt.Sprint(limit)
	return ms.list_members_by_query(less, query)
}
