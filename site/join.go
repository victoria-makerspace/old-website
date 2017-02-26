package site

import (
	"log"
)

//TODO: create talk user
func join_handler(p *page) {
	p.Name = "join"
	p.Title = "Join"
	p.authenticate()
	if p.Session != nil {
		p.http_error(403)
		return
	}
	p.ParseForm()
	if _, ok := p.PostForm["join"]; !ok {
		return
	}
	username := p.PostFormValue("username")
	email := p.PostFormValue("email")
	if available, err := p.Check_username_availability(username);
		!available {
		p.Data["username_error"] = err
		return
	}
	if available, err := p.Check_email_availability(email);
		!available {
		p.Data["email_error"] = err
		return
	}
	m := p.New_member(username, p.PostFormValue("name"), email,
		p.PostFormValue("password"))
	if m != nil {
		log.Panicf("Join error with username %s, email %s\n", username, email)
		return
	}
	//TODO: only sign-in and create talk user once email has been verified
	p.new_session(m, true)
	m.Sync(m.Id, m.Username, m.Email, m.Name)
}
