package site

import (
	"database/sql"
	"log"
	"net/http"
	"time"
)

func (m *member) update_membership_rate(db *sql.DB) {
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM student WHERE username = $1", m.Username).Scan(&n)
	if err != nil {
		log.Panic(err)
	}
	if m.Student != nil {
		if n == 1 {
			m.Student = &student{}
			_, err = db.Exec("UPDATE student SET institution = $2, graduation_date = $3 WHERE username = $1", m.Username, m.Student.Institution, m.Student.Grad_date)
		} else {
			_, err = db.Exec("INSERT INTO student (username, institution, graduation_date) VALUES ($1, $2, $3)", m.Username, m.Student.Institution, m.Student.Grad_date)
		}
		if err != nil {
			log.Panic(err)
		}
		return
	} else if n == 1 {
		_, err = db.Exec("DELETE FROM student WHERE username = $1", m.Username)
		if err != nil {
			log.Panic(err)
		}
	}
}

func (s *Http_server) billing_handler() {
	s.mux.HandleFunc("/member/billing", func(w http.ResponseWriter, r *http.Request) {
		s.parse_templates()
		p := page{Name: "billing", Title: "Billing"}
		s.authenticate(w, r, &p.Member)
		if !p.Member.Authenticated() {
			http.Error(w, http.StatusText(403), 403)
			return
		}
		p.Member.Billing = s.billing.Get_profile(p.Member.Username)
		r.ParseForm()
		if token := r.PostFormValue("singleUseToken"); token != "" {
			if p.Member.Billing != nil {
				p.Member.Billing.Update_card(r.PostFormValue("name"), token)
			} else {
				p.Member.Billing = s.billing.New_profile(token, p.Member.Name, p.Member.Username)
			}
			http.Redirect(w, r, "/member/billing", 303)
		} else if _, ok := r.PostForm["delete-card"]; ok && p.Member.Billing != nil {
			p.Member.Billing.Delete_card()
		}
		if _, ok := r.PostForm["register"]; ok {
			if r.PostFormValue("rate") == "student" && r.PostFormValue("institution") != "" && r.PostFormValue("graduation") != "" {
				graduation, err := time.Parse("2006-01", r.PostFormValue("graduation"))
				if err == nil && graduation.After(time.Now().AddDate(0, 1, 0)) {
					p.Member.Student = &student{r.PostFormValue("institution"), graduation}
				}
			}
			p.Member.update_membership_rate(s.db)
			//ip := r.RemoteAddr[:strings.LastIndexByte(r.RemoteAddr, ':')]
			amount := 50.00
			if p.Member.Student != nil {
				amount = 30.00
			}
			p.Member.Billing.Update_billing("Membership dues", amount)
			http.Redirect(w, r, "/member/billing", 303)
		} else {
			p.Member.get_student(s.db)
		}
		s.tmpl.Execute(w, p)
	})
}
