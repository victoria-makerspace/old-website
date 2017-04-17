package site

import (
	"fmt"
	"log"
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
	req_payload := p.Talk_api.Parse_sso_req(p.URL.Query())
	if req_payload != nil {
		return_path = req_payload.Get("return_sso_url")
		if return_path == "" {
			return_path = p.Talk_api.Path + "/session/sso_login"
		}
		values := url.Values{}
		values.Set("external_id", fmt.Sprint(p.Member.Id))
		values.Set("email", p.Member.Email)
		values.Set("nonce", req_payload.Get("nonce"))
		rsp_payload, rsp_sig := p.Talk_api.Encode_sso_rsp(values)
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
	m.Send_password_reset()
	p.Data["reset_send_success"] = true
}

//TODO: Move initial verification to join page
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
		if err := p.Verify_email(email); err != nil {
			//TODO: determine whether the server failed or discourse rejected
			//	e-mail address
			p.Data["server_error"] = true
			log.Println(err)
			return
		}
		p.redirect = "/sso?username=" + url.QueryEscape(p.Username)
		return
	}
	if _, ok := p.PostForm["send-verification-email"]; !ok {
		return
	}
	if p.Email == p.PostFormValue("email") {
		p.Data["email_error"] = "E-mail address already verified"
		return
	} else if !p.Email_available(p.PostFormValue("email")) {
		p.Data["email_error"] = "E-mail address is already in use"
		return
	} else if !p.Authenticate(p.PostFormValue("password")) {
		p.Data["password_error"] = "Incorrect password"
		return
	}
	p.Form.Add("sent", "true")
	member.Send_email_verification(p.PostFormValue("email"), p.Member)
	return
}
