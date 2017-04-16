package site

import (
	"regexp"
	"strconv"
)

func init() {
	init_handler("dashboard", dashboard_handler, "/member/dashboard")
	init_handler("account", account_handler, "/member/account")
	init_handler("profile", profile_handler, "/member/")
}

func dashboard_handler(p *page) {
	p.Title = "Dashboard"
	if !p.must_authenticate() {
		return
	}
}

func account_handler(p *page) {
	p.Title = "Account"
	if !p.must_authenticate() {
		return
	}
	if _, ok := p.PostForm["update-password"]; ok {
		if !p.Member.Authenticate(p.PostFormValue("old-password")) {
			p.Data["old_password_error"] = "Incorrect password"
			return
		}
		if p.PostFormValue("new-password") == "" {
			p.Data["new_password_error"] = "Password cannot be blank"
			return
		}
		p.Set_password(p.PostFormValue("new-password"))
		p.Data["update_password_success"] = "Successfully updated password"
		return
	} else if name := p.PostFormValue("name"); name != "" {
		if err := p.Set_name(name); err != nil {
			p.Data["name_error"] = err
		}
	} else if tel := p.PostFormValue("telephone"); tel != "" {
		if err := p.Set_telephone(tel); err != nil {
			p.Data["telephone_error"] = err
		}
	}
}

var profile_path_rexp = regexp.MustCompile(`^/member/[0-9]+$`)

func profile_handler(p *page) {
	if !profile_path_rexp.MatchString(p.URL.Path) {
		p.http_error(404)
	}
	member_id, _ := strconv.Atoi(p.URL.Path[len("/member/"):])
	m := p.Get_member_by_id(member_id)
	if m == nil {
		p.http_error(404)
		return
	}
	p.Title = "@" + m.Username
	p.Data["member"] = m
}
