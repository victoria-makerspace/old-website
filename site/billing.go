package site

import (
	"net/http"
	"strconv"
	"time"
)

func (h *Http_server) billing_handler() {
	h.mux.HandleFunc("/member/billing", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("billing", "Billing", w, r)
		if !p.must_authenticate() {
			return
		}
		pay_profile := p.Payment()
		write_template := func(p *page) {
			if pay_profile != nil {
				if card := pay_profile.Get_card(); card != nil {
					p.Field["card_number"] = card.Number
					p.Field["card_expiry"] = card.ExpiryMonth + "/20" + card.ExpiryYear
				}
			}
			p.write_template()
		}
		if !p.Member().Agreed_to_terms {
			write_template(p)
			return
		}
		p.ParseForm()
		if token := p.PostFormValue("singleUseToken"); token != "" {
			pay_profile.Update_card(token, p.PostFormValue("name"))
			http.Redirect(w, r, "/member/billing", 303)
			return
		} else if _, ok := p.PostForm["delete-card"]; ok && pay_profile != nil {
			pay_profile.Delete_card()
		}
		update_student := func() {
			if p.PostFormValue("rate") == "student" &&
				p.PostFormValue("institution") != "" &&
				p.PostFormValue("student_email") != "" &&
				p.PostFormValue("graduation") != "" {
				graduation, err := time.Parse("2006-01", p.PostFormValue("graduation"))
				if err == nil && graduation.After(time.Now().AddDate(0, 1, 0)) {
					p.Update_student(p.PostFormValue("institution"),
						p.PostFormValue("student_email"), graduation)
				} else {
					//TODO: embed error in page
				}
			} else {
				p.Delete_student()
			}
		}
		if _, ok := p.PostForm["update"]; ok {
			update_student()
			http.Redirect(w, r, "/member/billing", 303)
			return
		} else if _, ok := p.PostForm["register"]; ok {
			update_student()
			if p.Member().Active() {
				p.http_error(422)
				return
			}
			p.New_membership()
			http.Redirect(w, r, "/member/billing", 303)
			return
		} else if _, ok := p.PostForm["terminate"]; ok {
			id, _ := strconv.Atoi(p.PostFormValue("terminate"))
			if bill := pay_profile.Get_bill(id); bill != nil {
				pay_profile.Cancel_recurring_bill(bill)
			}
			// Redirect not really necessary as double-submission is harmless.
		} else if _, ok := p.PostForm["terminate_membership"]; ok {
			if !p.Member().Authenticate(p.PostFormValue("password")) {
				//TODO: embed error
				write_template(p)
				return
			}
			//TODO: reason for cancellation: PostFormValue("cancellation_reason")
			pay_profile.Cancel_membership()
			http.Redirect(w, r, "/member/billing", 303)
			return
		}
		write_template(p)
	})
}
