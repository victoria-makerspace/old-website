package site

import ()

func init() {
	handlers["/member/dashboard"] = member_handler
	handlers["/member/account"] = account_handler
	handlers["/tools"] = tools_handler
	handlers["/member/storage"] = storage_handler
}

func member_handler(p *page) {
	p.Name = "dashboard"
	p.Title = "Dashboard"
	if !p.must_authenticate() {
		return
	}
}

func account_handler(p *page) {
	p.Name = "account"
	p.Title = "Account"
	if !p.must_authenticate() {
		return
	}
	if _, ok := p.PostForm["update-password"]; ok {
		if !p.Authenticate(p.PostFormValue("old-password")) {
			p.Data["old_password_error"] = "Incorrect password"
			return
		}
		if p.PostFormValue("new-password") == "" {
			p.Data["new_password_error"] = "Password cannot be blank"
			return
		}
		p.Change_password(p.PostFormValue("new-password"))
		p.Data["update_password_success"] = "Successfully updated password"
		return
	}
}

func tools_handler(p *page) {
	p.Name = "tools"
	p.Title = "Tools"
	if !p.must_authenticate() {
		return
	}
}

func storage_handler(p *page) {
	p.Name = "storage"
	p.Title = "Storage"
	if !p.must_authenticate() {
		return
	}
	p.Data["wall_storage"] = p.Get_storage(p.Find_fee("storage", "wall"))
	p.Data["hall_lockers"] = p.Get_storage(p.Find_fee("storage", "hall-locker"))
	p.Data["bathroom_lockers"] = p.Get_storage(p.Find_fee("storage", "bathroom-locker"))
}
