package site

import (
	"log"
	"net/url"
)

func init() {
	init_handler("join", join_handler, "/join")
}

func join_handler(p *page) {
	p.Title = "Join"
	if p.Session != nil {
		p.http_error(403)
		return
	}
	if t := p.FormValue("token"); t != "" {
		if email, member := p.Verify_email_token(t); email != "" {
			if member != nil {
				p.http_error(400)
				return
			}
			p.Data["email"] = email
			return
		}
		p.Data["token_error"] = "Invalid verification token"
		return
	}
	if _, ok := p.PostForm["join"]; !ok {
		return
	}
	m, err := p.New_member(p.PostFormValue("username"), p.PostFormValue("name"),
		p.PostFormValue("email"))
	if err != nil {
		p.Data["join_error"] = err
		return
	}
	m.Set_password(p.PostFormValue("password"))
	log.Printf("New member: (%d) %s <%s>\n", m.Id, m.Username, m.Email)
	p.redirect = "/sso?username=" + url.QueryEscape(m.Username)
}
