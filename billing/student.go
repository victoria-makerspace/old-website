package billing

import (
	"github.com/lib/pq"
	"log"
	"database/sql"
	"time"
)

type student struct {
	Institution     string
	Email           string
	Graduation_date time.Time
}

func get_student(username string, db *sql.DB) *student {
	var (
		institution, email sql.NullString
		grad_date pq.NullTime
	)
	if err := db.QueryRow("SELECT institution, student_email, graduation_date FROM student WHERE username = $1", username).Scan(&institution, &email, &grad_date); err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return nil
	}
	return &student{institution.String, email.String, grad_date.Time}
}

