package site

import (
	"fmt"
	"github.com/vvanpo/makerspace/member"
	"net/url"
)

func init() {
	init_handler("sso", sso_handler, "/sso")
	init_handler("sign-out", sso_sign_out_handler, "/sso/sign-out")
	init_handler("reset-password", sso_reset_handler, "/sso/reset")
	init_handler("verify-email", sso_verify_email_handler, "/sso/verify-email")
}

func (p *page) must_authenticate() bool {
	if p.Session == nil {
		p.tmpl = handlers["sso"].Template
		p.Title = "Sign-in"
		p.Status = 403
		p.Data["return_path"] = p.URL.String()
		return false
	}
	return true
}

// sso_handler handles sign-in requests from the talk server, as well as serving
//	local sign-in requests/responses
func sso_handler(p *page) {
	p.Title = "Sign-in"
	return_path := "/member/dashboard"
	if rp, ok := p.Data["return_path"].(string); ok {
		return_path = rp
	} else if rp := p.PostFormValue("return_path"); rp != "" {
		return_path = rp
	}
	if p.Session == nil {
		// Embeds return_path in the sign-in form
		p.Data["return_path"] = return_path
		if p.FormValue("sso") != "" && p.FormValue("sig") != "" {
			p.Data["sso"] = p.FormValue("sso")
			p.Data["sig"] = p.FormValue("sig")
		}
		if _, ok := p.PostForm["sign-in"]; !ok {
			p.Data["username"] = p.FormValue("username")
			return
		}
		m := p.Get_member_by_username(p.PostFormValue("username"))
		if m == nil {
			p.Data["error_username"] = "Invalid username"
			return
		} else if !m.Authenticate(p.PostFormValue("password")) {
			p.Data["username"] = m.Username
			p.Data["error_password"] = "Incorrect password"
			return
		}
		p.new_session(m, !(p.PostFormValue("save-session") == "on"))
		if p.Session == nil {
			p.http_error(500)
			return
		}
	}
	// Won't reach this point without a session
	req_payload := p.Talk.Parse_sso_req(p.URL.Query())
	if req_payload != nil {
		return_path = req_payload.Get("return_sso_url")
		if return_path == "" {
			return_path = p.Talk.Path + "/session/sso_login"
		}
		values := url.Values{}
		values.Set("external_id", fmt.Sprint(p.Member.Id))
		values.Set("username", p.Member.Username)
		values.Set("name", p.Member.Name)
		values.Set("email", p.Member.Email)
		values.Set("nonce", req_payload.Get("nonce"))
		rsp_payload, rsp_sig := p.Talk.Encode_sso_rsp(values)
		return_path += "?sso=" + rsp_payload + "&sig=" + rsp_sig
	}
	p.redirect = return_path
}

func sso_sign_out_handler(p *page) {
	return_path := "/"
	if u := p.FormValue("return_path"); u != "" {
		return_path = u
	}
	if p.must_authenticate() {
		//TODO: find a secure way to to sign out that works with discourse
		if t := p.Talk_user(); t != nil {
			t.Logout()
		}
		p.destroy_session()
	}
	p.redirect = return_path
}

func sso_reset_handler(p *page) {
	p.Title = "Reset password"
	if p.Session != nil {
		p.http_error(403)
		return
	}
	p.Data["username"] = p.FormValue("username")
	p.Data["email"] = p.FormValue("email")
	if token := p.FormValue("token"); token != "" {
		p.Data["token"] = token
		m, err := p.Get_member_from_reset_token(token)
		if err != nil {
			p.Data["token_error"] = err
		} else if password := p.PostFormValue("password"); password != "" {
			m.Set_password(password)
			p.redirect = "/sso"
		}
		return
	}
	if _, ok := p.PostForm["reset-password"]; !ok {
		return
	}
	m := p.Get_member_by_username(p.PostFormValue("username"))
	if m == nil {
		p.Data["username_error"] = "Invalid username"
		delete(p.Data, "username")
		return
	} else if p.PostFormValue("email") != m.Email {
		p.Data["email_error"] = "Incorrect E-mail address for " +
			p.PostFormValue("username")
		delete(p.Data, "email")
		return
	}
	m.Send_password_reset(p.Config.Url())
	p.Data["reset_send_success"] = true
}

func sso_verify_email_handler(p *page) {
	if !p.must_authenticate() {
		return
	}
	p.Title = "Verify e-mail address"
	if token := p.FormValue("token"); token != "" {
		email, m := p.Verify_email_token(token)
		if email == "" {
			p.Data["token_error"] = "Invalid verification token"
			return
		}
		if m.Id != p.Member.Id {
			p.http_error(403)
			return
		}
		if err := p.Update_email(email); err != nil {
			p.Data["email_error"] = err
			return
		}
		p.redirect = "/member/account"
		return
	}
	email := p.FormValue("email")
	if p.Email == email {
		p.Data["email_error"] = "E-mail address already verified"
		return
	} else if err := member.Validate_email(email); err != nil {
		p.Data["email_error"] = err
		return
	} else if !p.Email_available(email) {
		p.Data["email_error"] = "E-mail address is already in use"
		return
	}
	if _, ok := p.PostForm["send-verification-email"]; !ok {
		return
	} else if !p.Authenticate(p.PostFormValue("password")) {
		p.Data["password_error"] = "Incorrect password"
		return
	}
	p.Form.Add("sent", "true")
	message := "Hello " + p.Member.Name + " (@" + p.Member.Username + "),\n\n" +
		"To change the e-mail address associated with your Makerspace "+
		"account (" + p.Member.Email + "), you must first verify that you " +
		"are its owner.\n\n" +
		"If the above name and username is correct, please verify your " +
		"e-mail address (" + email + ") by visiting " +
		p.Config.Url() + "/sso/verify-email?token="
	p.Send_email_verification(email, message, p.Member)
	return
}
