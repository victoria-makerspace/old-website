package site

import (
	"strconv"
)

func init() {
	init_handler("/admin", "admin", admin_handler)
}

func (p *page) must_be_admin() bool {
	if !p.must_authenticate() {
		return false
	} else if p.Admin == nil {
		p.http_error(403)
		return false
	}
	return true
}

func admin_handler(p *page) {
	p.Title = "Admin panel"
	if !p.must_be_admin() {
		return
	}
	if p.PostFormValue("approve_membership") != "" {
		member_id, err := strconv.Atoi(p.PostFormValue("approve_membership"))
		if err != nil {
			p.http_error(400)
			return
		}
		if member := p.Get_member_by_id(member_id); member != nil && !member.Approved {
			p.Member.Approve_member(member)
			p.Data["Member_approved"] = member
		} else {
			p.http_error(400)
			return
		}
	} else if p.PostFormValue("decline_membership") != "" {

	}
}
