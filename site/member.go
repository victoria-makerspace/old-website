package site

import (
	"github.com/vvanpo/makerspace/member"
	"regexp"
	"strconv"
	"time"
)

func init() {
	init_handler("member-json", member_json_handler, "/member.json")
	init_handler("dashboard", dashboard_handler, "/member/dashboard")
	init_handler("account", account_handler, "/member/account")
	init_handler("profile", profile_handler, "/member/")
	init_handler("access-card", access_form_handler, "/member/access-card")
}

// Search for members by id, username, email, or name
//	Could probably factor some stuff out here...
func member_json_handler(p *page) {
	p.srv_json = true
	ms := make(map[string]interface{})
	populate_json := func(m *member.Member) {
		data := make(map[string]interface{})
		data["id"] = m.Id
		data["registered"] = m.Registered
		data["admin"] = false
		if m.Admin != nil {
			data["admin"] = true
		}
		if t := m.Talk_user(); t != nil {
			user := make(map[string]interface{})
			user["id"] = t.Id
			user["username"] = t.Username
			user["title"] = t.Title
			data["talk-user"] = user
		}
		if a := m.Avatar_url(240); a != "" {
			data["avatar-url"] = a
		}
		if p.Session != nil {
			data["name"] = m.Name
			data["email"] = m.Email
			if p.Admin != nil {
				data["key-card"] = m.Key_card
				data["telephone"] = m.Telephone
				data["agreed-to-terms"] = m.Agreed_to_terms
				data["customer-id"] = m.Customer_id
			}
		}
		ms[m.Username] = data
	}
	if ids, ok := p.Form["id"]; ok {
		for _, id := range ids {
			i, err := strconv.Atoi(id)
			if err != nil {
				p.http_error(400)
				p.Data["error"] = "Invalid ID format: " + id
				return
			}
			if m := p.Get_member_by_id(i); m != nil {
				populate_json(m)
			}
		}
	}
	if usernames, ok := p.Form["username"]; ok {
		for _, username := range usernames {
			if err := p.Validate_username(username); err != nil {
				p.http_error(400)
				p.Data["error"] = err.Error()
				return
			}
			if m := p.Get_member_by_username(username); m != nil {
				populate_json(m)
			}
		}
	}
	if emails, ok := p.Form["email"]; ok {
		for _, email := range emails {
			if err := member.Validate_email(email); err != nil {
				p.http_error(400)
				p.Data["error"] = err.Error()
				return
			}
			if m := p.Get_member_by_email(email); m != nil {
				populate_json(m)
			}
		}
	}
	if names, ok := p.Form["name"]; ok {
		if p.Session == nil {
			p.http_error(403)
			p.Data["error"] = "Must be signed in to query by member names"
			return
		}
		for _, name := range names {
			if err := member.Validate_name(name); err != nil {
				p.http_error(400)
				p.Data["error"] = err.Error()
				return
			}
		}
		for _, m := range p.List_members_by_name(names) {
			populate_json(m)
		}
	}
	p.Data = ms
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
		if !p.Member.Authenticate(p.PostFormValue("current-password")) {
			p.Data["current_password_error"] = "Incorrect password"
			return
		}
		if p.PostFormValue("new-password") == "" {
			p.Data["new_password_error"] = "Password cannot be blank"
			return
		}
		p.Set_password(p.PostFormValue("new-password"))
		p.Data["update_password_success"] = "Successfully updated password"
		return
	} else if username := p.PostFormValue("username"); username != "" {
		if err := p.Update_username(username); err != nil {
			p.Data["username_error"] = err
		}
	} else if name := p.PostFormValue("name"); name != "" {
		if err := p.Update_name(name); err != nil {
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

func access_form_handler(p *page) {
	p.Title = "Access card request"
	if !p.must_authenticate() {
		return
	}
	if _, ok := p.PostForm["request-card"]; ok {
		var error bool
		if tel := p.PostFormValue("telephone"); tel == "" {
			p.Data["telephone_error"] = "Telephone number cannot be blank"
			error = true
		} else {
			err := p.Set_telephone(tel)
			if err != nil {
				p.Data["telephone_error"] = err
				error = true
			}
		}
		if p.PostFormValue("vehicle") != "" {
			err := p.Set_vehicle(p.PostFormValue("vehicle"))
			if err != nil {
				p.Data["vehicle_error"] = err
				error = true
			}
			if p.PostFormValue("plate") == "" {
				p.Data["plate_error"] = "Please submit your vehicle's license plate number"
				error = true
			}
		}
		if p.PostFormValue("plate") != "" {
			err := p.Set_license_plate(p.PostFormValue("plate"))
			if err != nil {
				p.Data["plate_error"] = err
				error = true
			}
			if p.PostFormValue("vehicle") == "" {
				p.Data["vehicle_error"] = "Please submit your vehicle's make and model"
				error = true
			}
		}
		if p.PostFormValue("agree-to-declaration") != "on" {
			p.Data["declaration_error"] = "You must agree to this declaration"
			error = true
		}
		if !error {
			p.Set_card_request_date(time.Now())
			p.redirect = "/member/billing"
		}
	}
}
