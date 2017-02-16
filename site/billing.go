package site

import (
	"net/http"
	"strconv"
	"time"
)

func (h *Http_server) billing_handler() {
	h.mux.HandleFunc("/member/billing", func(w http.ResponseWriter,
		r *http.Request) {
		p := h.new_page("billing", "Billing", w, r)
		p.authenticate()
		if p.Session == nil {
			p.http_error(403)
			return
		}
		pay_profile := p.billing.Get_profile(p.Member())
		p.ParseForm()
		if token := p.PostFormValue("singleUseToken"); token != "" {
			pay_profile.Update_card(p.PostFormValue("name"), token)
			http.Redirect(w, r, "/member/billing", 303)
		} else if _, ok := p.PostForm["delete-card"]; ok && pay_profile != nil {
			pay_profile.Delete_card()
		}
		if _, ok := p.PostForm["update"]; ok {
			if p.PostFormValue("rate") == "student" &&
				p.PostFormValue("institution") != "" &&
				p.PostFormValue("student_email") != "" &&
				p.PostFormValue("graduation") != "" {
				graduation, err := time.Parse("2006-01", p.PostFormValue("graduation"))
				if err == nil && graduation.After(time.Now().AddDate(0, 1, 0)) {
					if p.Member().Active && !p.Member().Student {
						//TODO: find invoice (regardless who's paying it) and update to student
					}
					p.Member().Update_student(p.PostFormValue("institution"),
						p.PostFormValue("student_email"), graduation)
					http.Redirect(w, r, "/member/billing", 303)
				} else {
					//TODO: embed error in page
				}
			} else {
				if p.Member().Active && p.Member().Student {
					//TODO: same as above, update invoice
				}
				p.Member().Delete_student()
				http.Redirect(w, r, "/member/billing", 303)
			}
		}
		if _, ok := p.PostForm["register"]; ok {
			if p.Member().Active {
				p.http_error(422)
				return
			}
			if pay_profile.Error == nil {
				//TODO: embed error response
				p.write_template()
				return
			}
			member_type := "membership_regular"
			if p.Member().Student {
				member_type = "membership_student"
			} else if p.Member().Corporate {
				//TODO
			}
			pay_profile.New_recurring_bill(
				pay_profile.billing.Fees[member_type].Id, p.Member().Username)
			http.Redirect(w, r, "/member/billing", 303)
		} else if _, ok := p.PostForm["terminate"]; ok {
			////////// TODO: password check
			id, _ := strconv.Atoi(p.PostFormValue("terminate"))
			if bill := pay_profile.Get_bill(id); bill != nil {
				bill.Cancel_recurring_bill()
			}
			// Redirect not really necessary as double-submission is harmless.
		}
		if p.Member().Student {
			student := p.Member().Get_student()
			p.Field["student_institution"] = student.Institution
			p.Field["student_email"] = student.Email
			p.Field["student_graduation_date"] = student.Graduation_date.Format("2006-01")
		}
		if pay_profile != nil {
			if card := pay_profile.Get_card(); card != nil {
				p.Field["card_number"] = card.Number
				p.Field["card_expiry"] = card.ExpiryMonth + "/20" + card.ExpiryYear
			}
			p.Field["pay_profile"] = pay_profile
		}
		p.write_template()
	})
}
