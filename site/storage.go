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
	if plan := p.PostFormValue("request-storage-plan"); plan != "" {
		if p.Get_payment_source() == nil {
			p.http_error(403)
			return
		}
		plan_id := "storage-" + member.Plan_identifier(plan)
		if err := p.Request_subscription(plan_id); err != nil {
			p.Data["request_storage_error"] = err
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
