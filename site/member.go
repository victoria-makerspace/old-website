package site

import (
	"github.com/vvanpo/makerspace/member"
	"regexp"
	"strconv"
)

func init() {
	init_handler("member-json", member_json_handler, "/member.json")
	init_handler("dashboard", dashboard_handler, "/member/dashboard")
	init_handler("account", account_handler, "/member/account")
	init_handler("profile", profile_handler, "/member/")
}

// Search for members by id, username, email, or name
func member_json_handler(p *page) {
	p.srv_json = true
	var (
		ids map[int]*member.Member
		usernames map[string]*member.Member
		emails map[string]*member.Member
		names map[string][]*member.Member
	)
	if is, ok := p.Form["id"]; ok {
		ids = make(map[int]*member.Member)
		for _, id := range is {
			i, err := strconv.Atoi(id)
			if err != nil {
				p.http_error(400)
				p.Data["error"] = "Invalid ID format: " + id
				return
			}
			ids[i] = p.Get_member_by_id(i)
		}
	}
	if us, ok := p.Form["username"]; ok {
		usernames = make(map[string]*member.Member)
		for _, username := range us {
			if err := p.Validate_username(username); err != nil {
				p.http_error(400)
				p.Data["error"] = err.Error()
				return
			}
			usernames[username] = p.Get_member_by_username(username)
		}
	}
	if es, ok := p.Form["email"]; ok {
		if p.Session == nil {
			p.http_error(403)
			p.Data["error"] = "Must be signed in to query by member e-mails"
			return
		}
		emails = make(map[string]*member.Member)
		for _, email := range es {
			if err := member.Validate_email(email); err != nil {
				p.http_error(400)
				p.Data["error"] = err.Error()
				return
			}
			emails[email] = p.Get_member_by_email(email)
		}
	}
	if ns, ok := p.Form["name"]; ok {
		if p.Session == nil {
			p.http_error(403)
			p.Data["error"] = "Must be signed in to query by member names"
			return
		}
		names = make(map[string][]*member.Member)
		for _, name := range ns {
			if err := member.Validate_name(name); err != nil {
				p.http_error(400)
				p.Data["error"] = err.Error()
				return
			}
			ms := p.Get_members_by_name(name)
			names[name] = make([]*member.Member, 0, len(ms))
			for _, m := range ms {
				names[name] = append(names[name], m)
			}
		}
	}
	populate_json := func(m *member.Member) map[string]interface{} {
		data := make(map[string]interface{})
		data["id"] = m.Id
		data["username"] = m.Username
		data["registered"] = m.Registered
		data["admin"] = false
		if m.Admin != nil {
			data["admin"] = true
		}
		if t := m.Talk_user(); t != nil {
			data["talk-id"] = t.Id
		}
		if a := m.Avatar_url(240); a != "" {
			data["avatar-url"] = a
		}
		if p.Session != nil {
			data["name"] = m.Name
			data["email"] = m.Email
		}
		if p.Admin != nil {
			data["key-card"] = m.Key_card
			data["telephone"] = m.Telephone
			data["agreed-to-terms"] = m.Agreed_to_terms
			data["customer-id"] = m.Customer_id
		}
		return data
	}
	if ids != nil {
		id_list := make(map[int]map[string]interface{})
		for i, m := range ids {
			id_list[i] = populate_json(m)
		}
		p.Data["ids"] = id_list
	}
	if usernames != nil {
		name_list := make(map[string]map[string]interface{})
		for u, m := range usernames {
			name_list[u] = populate_json(m)
		}
		p.Data["usernames"] = name_list
	}
	if emails != nil {
		email_list := make(map[string]map[string]interface{})
		for e, m := range emails {
			email_list[e] = populate_json(m)
		}
		p.Data["emails"] = email_list
	}
	if names != nil {
		name_list := make(map[string][]map[string]interface{})
		for n, ms := range names {
			name_list[n] = make([]map[string]interface{}, len(ms))
			for i, m := range ms {
				name_list[n][i] = populate_json(m)
			}
		}
		p.Data["names"] = name_list
	}
}

func dashboard_handler(p *page) {
	p.Title = "Dashboard"
	if !p.must_authenticate() {
		return
	}
}

func account_handler(p *page) {
	p.Title = "Account"
	if !p.must_authenticate() {
		return
	}
	if _, ok := p.PostForm["update-password"]; ok {
		if !p.Member.Authenticate(p.PostFormValue("old-password")) {
			p.Data["old_password_error"] = "Incorrect password"
			return
		}
		if p.PostFormValue("new-password") == "" {
			p.Data["new_password_error"] = "Password cannot be blank"
			return
		}
		p.Set_password(p.PostFormValue("new-password"))
		p.Data["update_password_success"] = "Successfully updated password"
		return
	} else if name := p.PostFormValue("name"); name != "" {
		if err := p.Set_name(name); err != nil {
			p.Data["name_error"] = err
		}
	} else if tel := p.PostFormValue("telephone"); tel != "" {
		if err := p.Set_telephone(tel); err != nil {
			p.Data["telephone_error"] = err
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
	p.Data["member"] = m
}
