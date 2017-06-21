package site

import (
	"github.com/vvanpo/makerspace/member"
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
	if _, ok := p.PostForm["verify-email"]; ok {
		email := p.PostFormValue("email")
		if err := member.Validate_email(email); err != nil {
			p.Data["email_error"] = err
			return
		} else if !p.Email_available(email) {
			p.Data["email_error"] = "E-mail address is already in use"
			return
		}
		message := "Hello,\n\n" +
			"To register for a Makerspace account, you must first verify " +
			"the ownership of this e-mail address.\n\n" +
			"Please verify your e-mail address (" + email + ") by visiting " +
			p.Config.Url() + "/join?token="
		p.Send_email_verification(email, message, nil)
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
