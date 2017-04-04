package site

import ()

func init() {
	init_handler("/member/dashboard", "dashboard", dashboard_handler)
	init_handler("/member/account", "account", account_handler)
	init_handler("/member/storage", "storage", storage_handler)
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
