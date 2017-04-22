package site

import (
	"strconv"
)

func init() {
	init_handler("storage", storage_handler, "/member/storage")
}

func storage_handler(p *page) {
	p.Title = "Storage"
	if !p.must_authenticate() {
		return
	}
	if plan := p.PostFormValue("register-storage-plan"); plan != "" {
		if !p.Has_card() {
			p.http_error(403)
			return
		}
		number, err := strconv.Atoi(p.PostFormValue("register-storage-number"))
		if err != nil {
			p.http_error(400)
			return
		}
		if err := p.New_storage_lease(plan, number); err != nil {
			p.Data["register_storage_error"] = err
		} else {
			p.redirect = "/member/storage"
		}
	}
}
