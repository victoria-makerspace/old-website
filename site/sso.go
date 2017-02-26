package site

import (
	"fmt"
	"net/url"
)

func init() {
	handlers["/sso"] = sso_handler
	handlers["/sso/sign-out"] = sso_sign_out_handler
	handlers["/sso/check-availability.json"] = sso_availability_handler
}

func (p *page) must_authenticate() bool {
	p.authenticate()
	if p.Session == nil {
		p.Name = "sso"
		p.Title = "Sign-in"
		p.Status = 403
		return false
	}
	return true
}

// sso_handler handles sign-in requests from the talk server, as well as serving
//	local sign-in requests/responses
func sso_handler(p *page) {
	p.Name = "sso"
	p.Title = "Sign-in"
	p.authenticate()
	return_path := "/member/dashboard"
	if rp := p.PostFormValue("return_path"); rp != "" {
		return_path = rp
	}
	req_payload := p.Talk_api.Parse_sso_req(p.URL.Query())
	if req_payload != nil {
		if rp := req_payload.Get("return_sso_url"); rp != "" {
			return_path = rp
		}
	}
	if p.Session == nil {
		p.ParseForm()
		// Embeds return_path in the sign-in form
		p.Data["return_path"] = return_path
		if _, ok := p.PostForm["sign-in"]; !ok {
			return
		}
		m := p.Get_member_by_username(p.PostFormValue("username"))
		if m == nil {
			p.Data["error_username"] = "Invalid username"
			return
		} else if !m.Authenticate(p.PostFormValue("password")) {
			p.Data["error_password"] = "Incorrect password"
			return
		}
		p.new_session(m, !(p.PostFormValue("save-session") == "on"))
	}
	// Won't reach this point without a successful login
	if req_payload != nil {
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

func sso_availability_handler(p *page) {
	p.srv_json = true
	if u := p.FormValue("username"); u != "" {
		available, err := p.Check_username_availability(u)
		p.Data["username"] = available
		p.Data["username_error"] = err
	}
	if e := p.FormValue("email"); e != "" {
		available, err := p.Check_email_availability(e)
		p.Data["email"] = available
		p.Data["email_error"] = err
	}
}
