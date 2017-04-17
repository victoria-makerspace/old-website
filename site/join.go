package site

import (
	"log"
	"net/url"
	"github.com/vvanpo/makerspace/member"
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
	if _, ok := p.PostForm["verify-email"]; ok {
		if err := member.Validate_email(p.PostFormValue("email")); err != nil {
			p.Data["email_error"] = err
			return
		} else if !p.Email_available(p.PostFormValue("email")) {
			p.Data["email_error"] = "E-mail address is already in use"
			return
		}
		p.Send_email_verification(p.PostFormValue("email"), nil)
		p.Data["success"] = true
		return
	}
	token := p.FormValue("token")
	if token == "" {
		return
	}
	email, member := p.Verify_email_token(token)
	if email == "" {
		p.Data["token_error"] = "Invalid verification token"
		p.Form["token"] = nil
		return
	} else if member != nil {
		p.Data["token_error"] = "E-mail already in use"
		p.Form["token"] = nil
		p.Delete_verification_tokens(email)
		return
	}
	p.Data["email"] = email
	if _, ok := p.PostForm["join"]; !ok {
		return
	}
	m, err := p.New_member(p.PostFormValue("username"), p.PostFormValue("name"),
		email)
	if err != nil {
		p.Data["join_error"] = err
		return
	}
	p.Delete_verification_tokens(email)
	m.Set_password(p.PostFormValue("password"))
	log.Printf("New member: (%d) %s <%s>\n", m.Id, m.Username, m.Email)
	p.redirect = "/sso?username=" + url.QueryEscape(m.Username)
}
