package site

import ()

func init() {
	init_handler("dashboard", dashboard_handler, "/member/dashboard")
	init_handler("account", account_handler, "/member/account")
	init_handler("storage", storage_handler, "/member/storage")
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

func storage_handler(p *page) {
	p.Title = "Storage"
	if !p.must_authenticate() {
		return
	}
	p.Data["wall_storage"] = p.Get_storage(p.Find_fee("storage", "wall"))
	p.Data["hall_lockers"] = p.Get_storage(p.Find_fee("storage", "hall-locker"))
	p.Data["bathroom_lockers"] = p.Get_storage(p.Find_fee("storage", "bathroom-locker"))
}
