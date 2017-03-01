package site

import ()

func init() {
	handlers["/member/dashboard"] = member_handler
	handlers["/member/preferences"] = preferences_handler
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

func preferences_handler(p *page) {
	p.Name = "preferences"
	p.Title = "Preferences"
	if !p.must_authenticate() {
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

//preferences
