package site

import (
	"regexp"
	"strconv"
)

func init() {
	init_handler("member-list", member_list_handler, "/member/list",
		"/member/list/")
	init_handler("profile", profile_handler, "/member/")
}

func member_list_handler(p *page) {
	switch p.URL.Path {
	default:
		p.http_error(404)
	case "/member/list":
		p.Title = "All members"
		p.Data["member_group"] = "all"
		p.Data["member_list"] = p.Get_all_members()
	case "/member/list/active":
		p.Title = "Active members"
		p.Data["member_group"] = "active"
		p.Data["member_list"] = p.Get_all_active_members()
	case "/member/list/new":
		p.Title = "New members"
		p.Data["member_group"] = "new"
		limit := 20
		if v := p.FormValue("limit"); v != "" {
			if lim, err := strconv.Atoi(v); err == nil {
				limit = lim
			}
		}
		p.Data["member_list"] = p.Get_new_members(limit)
	case "/member/list/unapproved":
		if !p.must_be_admin() {
			return
		}
		p.Title = "Unapproved members"
		p.Data["member_group"] = "unapproved"
		p.Data["member_list"] = p.Get_all_unapproved_members()
	case "/member/list/pending":
		if !p.must_be_admin() {
			return
		}
		p.Title = "Pending-approval members"
		p.Data["member_group"] = "pending"
		p.Data["member_list"] = p.Get_all_pending_members()
	case "/member/list/unverified":
		if !p.must_be_admin() {
			return
		}
		p.Title = "Unverified members"
		p.Data["member_group"] = "unverified"
		p.Data["member_list"] = p.Get_all_unverified_members()
	}
}

var profile_path_rexp = regexp.MustCompile(`^/member/[0-9]+$`)

func profile_handler(p *page) {
	if !profile_path_rexp.MatchString(p.URL.Path) {
		p.http_error(404)
	}
	member_id, _ := strconv.Atoi(p.URL.Path[len("/member/"):])
	m := p.Get_member_by_id(member_id)
	if m == nil {
		p.http_error(404)
		return
	}
	p.Title = "@" + m.Username
	p.Data["member"] = m
}
