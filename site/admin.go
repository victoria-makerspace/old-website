package site

import ()

func init() {
	handlers["/admin"] = admin_handler
}

func (p *page) must_be_admin() bool {
	if !p.must_authenticate() {
		return false
	}
	if p.Admin == nil {
		return false
	}
	return true
}

func admin_handler(p *page) {
	p.Name = "admin"
	p.Title = "Admin panel"
	if !p.must_be_admin() {
		return
	}
}
