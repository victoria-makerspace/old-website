package site

import (
	"log"
	"net/url"
)

func init() {
	init_handler("/join", "join", join_handler)
}

func join_handler(p *page) {
	p.Title = "Join"
	if p.Session != nil {
		p.http_error(403)
		return
	}
	if _, ok := p.PostForm["join"]; !ok {
		return
	}
	username := p.PostFormValue("username")
	email := p.PostFormValue("email")
	name := p.PostFormValue("name")
	p.Data["username"] = username
	p.Data["email"] = email
	p.Data["name"] = name
	//TODO: don't create talk user until email has
	//	been verified, and delete all accounts with pending e-mail verifications
	//	for the same e-mail when one account verifies that e-mail address.
	m, err := p.New_member(username, email, name, p.PostFormValue("password"))
	for k, v := range err {
		p.Data[k] = v
	}
	if m == nil {
		log.Printf("Failed to create member: %s <%s>\n", username, email)
		return
	}
	log.Printf("New member: (%d) %s\n", m.Id, username)
	p.redirect = "/sso/verify-email?sent=true&username=" +
		url.QueryEscape(username) + "&email=" + url.QueryEscape(email)
}
