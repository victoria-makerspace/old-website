package site

import (
	"regexp"
	"strconv"
)

func init() {
	init_handler("member-list", member_list_handler, "/member/list",
		"/member/list/active")
	init_handler("profile", profile_handler, "/member/")
}

func member_list_handler(p *page) {
	switch p.URL.Path {
	default: p.http_error(404)
	case "/member/list/active": {
		p.Name = "active"
		p.Title = "Active members"
		p.Data["member_list"] = p.Get_all_approved_members()
	}
	case "/member/list": {
		p.Name = "all"
		p.Title = "All members"
		p.Data["member_list"] = p.Get_all_members()
	}
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
}
