package site

import (
	"strconv"
	"time"
)

func init() {
	handlers["/member/billing"] = billing_handler
}

func billing_handler(p *page) {
	p.Name = "billing"
	p.Title = "Billing"
	if !p.must_authenticate() {
		return
	}
	if !p.Agreed_to_terms {
		return
	}
	pay_profile := p.Payment()
	if token := p.PostFormValue("singleUseToken"); token != "" {
		if pay_profile == nil {
			pay_profile = p.New_profile(p.Member.Id)
		}
		pay_profile.Update_card(token, p.PostFormValue("name"))
		p.redirect = "/member/billing"
		return
	} else if _, ok := p.PostForm["delete-card"]; ok && pay_profile != nil {
		pay_profile.Delete_card()
		return
	}
	update_student := func() {
		if p.PostFormValue("rate") == "student" &&
			p.PostFormValue("institution") != "" &&
			p.PostFormValue("student_email") != "" &&
			p.PostFormValue("graduation") != "" {
			graduation, err := time.Parse("2006-01",
				p.PostFormValue("graduation"))
			if err == nil && graduation.After(time.Now().AddDate(0, 1, 0)) {
				p.Update_student(p.PostFormValue("institution"),
					p.PostFormValue("student_email"), graduation)
			} else {
				p.Data["graduation_error"] = "Graduation date cannot be in the past"
			}
		} else {
			p.Delete_student()
		}
	}
	if _, ok := p.PostForm["update"]; ok {
		update_student()
		p.redirect = "/member/billing"
		return
	} else if _, ok := p.PostForm["terminate_membership"]; ok {
		if !p.Member.Authenticate(p.PostFormValue("password")) {
			p.Data["password_error"] = "Incorrect password"
			return
		}
		//TODO: reason for cancellation: PostFormValue("cancellation_reason")
		p.Member.Cancel_membership()
		p.redirect = "/member/billing"
		return
	} else if pay_profile == nil {
		return
	} else if _, ok := p.PostForm["retry-missed-payments"]; ok {
		pay_profile.Retry_missed_payments()
		return
	} else if _, ok := p.PostForm["register"]; ok {
		update_student()
		if p.Member.Membership_invoice != nil {
			p.http_error(422)
			return
		}
		p.New_membership_invoice()
		p.redirect = "/member/billing"
		return
	} else if _, ok := p.PostForm["terminate"]; ok {
		id, _ := strconv.Atoi(p.PostFormValue("terminate"))
		if bill := pay_profile.Get_bill(id); bill != nil {
			pay_profile.Cancel_recurring_bill(bill)
		}
		// Redirect not really necessary as double-submission is harmless.
	}
}
