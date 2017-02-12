package member

import (
	"database/sql"
	"github.com/lib/pq"
	"log"
	"time"
)

type Student struct {
	Institution     string
	Email           string
	Graduation_date time.Time
}

func (m *Member) Get_student() *Student {
	var (
		institution, email sql.NullString
		grad_date          pq.NullTime
	)
	if err := m.db.QueryRow("SELECT institution, student_email, "+
		"graduation_date FROM student WHERE username = $1", m.Username).
		Scan(&institution, &email, &grad_date); err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return nil
	}
	return &Student{institution.String, email.String, grad_date.Time}
}

func (m *Member) Update_student(institution, email string, grad_date time.Time) {
	var is_student bool
	if err := m.db.QueryRow("SELECT true FROM student WHERE username = $1",
		m.Username).Scan(&is_student); err != nil {
		log.Panic(err)
	}
	query := "INSERT INTO student (username, institution, email, "+
		"graduation_date) VALUE ($1, $2, $3, $4)"
	if is_student {
		query = "UPDATE student SET institution = $2, student_email = $3, "+
			"graduation_date = $4 WHERE username = $1"
	}
	if _, err := m.db.Exec(query, m.Username, institution, email, grad_date);
		err != nil {
		log.Panic(err)
	}
	m.Student = true
}

func (m *Member) Delete_student() {
	if _, err := m.db.Exec("DELETE FROM student WHERE username = $1",
		m.Username); err != nil {
		log.Panic(err)
	}
	m.Student = false
}
