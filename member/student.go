package member

import (
	"fmt"
	"log"
	"time"
)

type Student struct {
	Institution     string
	Email           string
	Graduation_date time.Time
}

//TODO: verify student email
func (m *Member) Update_student(institution, email string, grad_date time.Time) error {
	if grad_date.Before(time.Now().AddDate(0, 1, 0)) {
		return fmt.Errorf("Graduation date cannot be in the past")
	}
	if !email_rexp.MatchString(email) {
		return fmt.Errorf("Invalid E-mail address")
	}
	query := "INSERT INTO student (member, institution, student_email, " +
		"graduation_date) VALUES ($1, $2, $3, $4)"
	if m.Student != nil {
		query = "UPDATE student SET institution = $2, student_email = $3, " +
			"graduation_date = $4 WHERE member = $1"
	}
	if _, err := m.Exec(query, m.Id, institution, email, grad_date); err != nil {
		log.Panic(err)
	}
	m.Student = &Student{institution, email, grad_date}
	return nil
}

func (m *Member) Delete_student() {
	if m.Student == nil {
		return
	}
	m.Student = nil
	if mp := m.Get_membership(); mp != nil &&
		mp.Plan.ID == "membership-student" {
		m.Update_membership("membership-regular")
	}
	if _, err := m.Exec("DELETE FROM student WHERE member = $1",
		m.Id); err != nil {
		log.Panic(err)
	}
}
