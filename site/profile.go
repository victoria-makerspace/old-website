package site

import (
	"regexp"
	"strconv"
)

func init() {
	init_handler("/member/list", "member-list", member_list_handler)
	init_handler("/member/", "profile", profile_handler)
}

func member_list_handler(p *page) {
	p.Title = "Members"
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
	}
	p.Title = "@" + m.Username
}

