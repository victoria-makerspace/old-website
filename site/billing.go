package site

import (
	_ "log"
	"net/http"
	"time"
)

func (p *page) get_student() {

}

func (p *page) update_membership_rate() {
	/*	var n int
		err := db.QueryRow("SELECT COUNT(*) FROM student WHERE username = $1", m.Username).Scan(&n)
		if err != nil {
			log.Panic(err)
		}
		if m.Student != nil {
			if n == 1 {
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
	*/
}

func (h *Http_server) billing_handler() {
	h.mux.HandleFunc("/member/billing", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("billing", "Billing", w, r)
		p.authenticate()
		if p.Session == nil {
			p.http_error(403)
			return
		}
		billing := p.billing.Get_profile(p.Member())
		p.Session.Billing = billing
		p.ParseForm()
		if token := p.PostFormValue("singleUseToken"); token != "" {
			if billing != nil {
				billing.Update_card(p.PostFormValue("name"), token)
			} else {
				billing = p.billing.New_profile(token, p.PostFormValue("name"), p.Member())
			}
			http.Redirect(w, r, "/member/billing", 303)
		} else if _, ok := p.PostForm["delete-card"]; ok && billing != nil {
			billing.Delete_card()
		}
		//p.Member().get_student(s.db)
		if _, ok := p.PostForm["register"]; ok {
			if p.PostFormValue("rate") == "student" && p.PostFormValue("institution") != "" && p.PostFormValue("graduation") != "" {
				graduation, err := time.Parse("2006-01", p.PostFormValue("graduation"))
				if err == nil && graduation.After(time.Now().AddDate(0, 1, 0)) {
					//p.Member.Student = &student{r.PostFormValue("institution"), graduation}
				} else {
					//p.Member.Student = nil
				}
			} else {
				//p.Member.Student = nil
			}
			/// TODO
			p.update_membership_rate()
			// TODO: update ip in transaction
			//ip := r.RemoteAddr[:strings.LastIndexByte(r.RemoteAddr, ':')]
			amount := 50.00
			// TODO
			/*if p.Member.Student != nil {
				amount = 30.00
			}*/
			p.Session.Billing.Update_billing("Membership dues", amount)
			http.Redirect(w, r, "/member/billing", 303)
		} else if _, ok := p.PostForm["terminate"]; ok {
			////////// TODO: password check
		}
		p.write_template()
	})
}
