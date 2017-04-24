package site

import (
	"github.com/vvanpo/makerspace/member"
	"log"
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
		if p.Get_payment_source() == nil {
			p.http_error(403)
			return
		}
		plan_id := "storage-" + member.Plan_identifier(plan)
		number, err := strconv.Atoi(p.PostFormValue("register-storage-number"))
		if err != nil {
			p.http_error(400)
			return
		}
		if err := p.New_storage_lease(plan_id, number); err != nil {
			p.Data["register_storage_error"] = err
		} else {
			p.redirect = "/member/storage"
		}
	} else if plan := p.PostFormValue("cancel-storage-plan"); plan != "" {
		number, err := strconv.Atoi(p.PostFormValue("cancel-storage-number"))
		if err != nil {
			p.http_error(400)
			return
		}
		plan_id := "storage-" + member.Plan_identifier(plan)
		if err := p.Cancel_storage_lease(plan_id, number); err != nil {
			log.Println("Cancel_storage_lease error: ", err)
			p.http_error(500)
			return
		} else {
			p.redirect = "/member/storage"
		}
	}
}
