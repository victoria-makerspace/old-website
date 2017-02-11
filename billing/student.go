package billing

import (
	"log"
	"time"
)

type student struct {
	Institution string
	Graduation_date time.Time
}

func (bp *Profile) Update_student(institution string, grad_date time.Time) {
	var query string
	if bp.Student != nil {
		query = "INSERT INTO student (username, institution, graduation_date) VALUE ($1, $2, $3)"
	}
	query = "UPDATE student SET institution = $2, graduation_date = $3 WHERE username = $1"
	if _, err := bp.db.Exec(query, bp.member.Username, institution, grad_date); err != nil {
		log.Panic(err)
	}
	bp.Student = &student{institution, grad_date}
}

func (bp *Profile) Delete_student() {
	if _, err := bp.db.Exec("DELETE FROM student WHERE username = $1", bp.member.Username); err != nil {
		log.Panic(err)
	}
}
