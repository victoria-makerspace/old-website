package site

import (
	"strconv"
)

func init() {
	init_handler("member-list", member_list_handler, "/member/list",
		"/member/list/")
}

func member_list_handler(p *page) {
	switch p.URL.Path {
	default:
		p.http_error(404)
	case "/member/list":
		p.Title = "All members"
		p.Data["member_group"] = "all"
		p.Data["member_list"] = p.List_members()
	case "/member/list/active":
		p.Title = "Active members"
		p.Data["member_group"] = "active"
		p.Data["member_list"] = p.List_active_members()
	case "/member/list/new":
		p.Title = "New members"
		p.Data["member_group"] = "new"
		limit := 20
		if v := p.FormValue("limit"); v != "" {
			if lim, err := strconv.Atoi(v); err == nil {
				limit = lim
			}
		}
		p.Data["member_list"] = p.List_new_members(limit)
	}
}
