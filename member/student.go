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

func (m *Member) get_student() {
	var institution, email sql.NullString
	var grad_date          pq.NullTime
	if err := m.QueryRow("SELECT institution, student_email, "+
		"graduation_date FROM student WHERE member = $1", m.Id).
		Scan(&institution, &email, &grad_date); err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return
	}
	m.Student = &Student{institution.String, email.String, grad_date.Time}
}

//TODO: verify student email
func (m *Member) Update_student(institution, email string, grad_date time.Time) {
	query := "INSERT INTO student (member, institution, student_email, " +
		"graduation_date) VALUES ($1, $2, $3, $4)"
	if m.Student != nil {
		query = "UPDATE student SET institution = $2, student_email = $3, " +
			"graduation_date = $4 WHERE member = $1"
	} else if m.membership != nil {
		m.payment.Change_to_student(grad_date)
	}
	if _, err := m.Exec(query, m.Id, institution, email, grad_date);
		err != nil {
		log.Panic(err)
	}
	m.Student = &Student{institution, email, grad_date}
}

func (m *Member) Delete_student() {
	m.Student = nil
	if _, err := m.Exec("DELETE FROM student WHERE member = $1",
		m.Id); err != nil {
		log.Panic(err)
	}
}
