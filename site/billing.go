package site

import (
	"time"
)

func init() {
	init_handler("billing", billing_handler, "/member/billing")
}

func billing_handler(p *page) {
	p.Title = "Billing"
	if !p.must_authenticate() {
		return
	}
	if token := p.PostFormValue("stripeToken"); token != "" {
		p.Update_customer(token, nil)
		p.redirect = "/member/billing"
		return
	}
	if _, ok := p.PostForm["membership"]; ok {
		if p.PostFormValue("rate") == "student" &&
			p.PostFormValue("institution") != "" &&
			p.PostFormValue("student_email") != "" &&
			p.PostFormValue("graduation") != "" {
			//TODO email regex and institution/email verification
			graduation, err := time.Parse("2006-01",
				p.PostFormValue("graduation"))
			if err != nil {
				p.http_error(400)
				return
			}
			if graduation.After(time.Now().AddDate(0, 1, 0)) {
				p.Update_student(p.PostFormValue("institution"),
					p.PostFormValue("student_email"), graduation)
			} else {
				p.Data["graduation_error"] = "Graduation date cannot be in the past"
				return
			}
		} else {
			p.Delete_student()
		}
		if _, ok := p.PostForm["register"]; ok {
			if err := p.Request_membership(); err != nil {
				p.Data["request_error"] = err
				return
			}
		}
	} else if _, ok := p.PostForm["terminate_membership"]; ok {
		if !p.Member.Authenticate(p.PostFormValue("password")) {
			p.Data["password_error"] = "Incorrect password"
			return
		}
		//TODO: reason for cancellation: PostFormValue("cancellation_reason")
		p.Member.Cancel_membership()
	} else if p.Customer() == nil || p.Customer().DefaultSource == nil || !p.Agreed_to_terms {
		return
	} else if _, ok := p.PostForm["terminate"]; ok {
		//id, _ := strconv.Atoi(p.PostFormValue("terminate"))
		//TODO terminate subscription
	}
	p.redirect = "/member/billing"
}
