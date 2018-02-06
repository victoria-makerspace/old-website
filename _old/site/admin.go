package site

import (
	"fmt"
	"github.com/vvanpo/makerspace/member"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func init() {
	init_handler("admin", admin_handler, "/admin")
	init_handler("admin-list", admin_list_handler, "/admin/list",
		"/admin/list/")
	init_handler("admin-manage", admin_account_handler, "/admin/account/")
	init_handler("admin-storage", admin_storage_handler, "/admin/storage")
	init_handler("admin-cards", admin_cards_handler, "/admin/access-cards")
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
	p.Title = "Administrator panel"
	if !p.must_be_admin() {
		return
	}
	if p.PostFormValue("approve-membership") != "" {
		member_id, err := strconv.Atoi(p.PostFormValue("approve-membership"))
		if err != nil {
			p.http_error(400)
			return
		}
		if m := p.Get_member_by_id(member_id); m == nil || m.Get_pending_membership() == nil {
			p.http_error(400)
			return
		} else {
			if err := p.Approve_membership(m); err != nil {
				p.http_error(500)
			}
			p.Data["Member_approved"] = m
			if m.Talk_user() != nil && p.PostFormValue("notify-member") == "on" {
				//TODO
			}
		}
	} else if p.PostFormValue("decline-membership") != "" {
		member_id, err := strconv.Atoi(p.PostFormValue("decline-membership"))
		if err != nil {
			p.http_error(400)
			return
		}
		m := p.Get_member_by_id(member_id)
		if m == nil {
			p.http_error(400)
			return
		}
		pending := m.Get_pending_membership()
		if pending == nil {
			p.http_error(400)
			return
		}
		p.Cancel_pending_subscription(pending)
		if m.Talk_user() != nil && p.PostFormValue("notify-member") == "on" {
			p.Talk.Message_user("Your membership request was declined",
				"Your membership request was declined by @"+p.Member.Username+
					".", m.Talk_user(), p.Member.Talk_user())
		}
	} else if p.PostFormValue("member-upload") != "" {
		member_upload_handler(p)
	}
	pending := p.List_all_pending_subscriptions()
	storage_req := make([]*member.Pending_subscription, 0)
	for i := 0; i < len(pending); i++ {
		if strings.HasPrefix(pending[i].Plan_id, "storage-") {
			storage_req = append(storage_req, pending[i])
			pending = append(pending[:i], pending[i+1:]...)
			i--
		}
	}
	p.Data["pending_subs"] = pending
	p.Data["storage_requests"] = storage_req
}

func admin_list_handler(p *page) {
	if !p.must_be_admin() {
		return
	}
	type member_list struct {
		Title     string
		Group     string
		Subgroups []member_list
		Members   func() []*member.Member
	}
	lists := []member_list{
		member_list{"members", "all", nil, p.List_members},
		member_list{"active members", "active", nil, p.List_active_members},
		member_list{"new members", "new", nil,
			func() []*member.Member {
				limit := 20
				if v := p.FormValue("limit"); v != "" {
					if lim, err := strconv.Atoi(v); err == nil {
						limit = lim
					}
				}
				return p.List_new_members(limit)
			}},
		member_list{"memberships", "approved", []member_list{
			member_list{"regular memberships", "regular", nil,
				func() []*member.Member {
					return p.List_members_with_membership("membership-regular")
				}},
			member_list{"student memberships", "student", nil,
				func() []*member.Member {
					return p.List_members_with_membership("membership-student")
				}},
			member_list{"free memberships", "free", nil,
				func() []*member.Member {
					return p.List_members_with_membership("membership-free")
				}},
		}, p.List_members_with_memberships},
	}
	p.Data["lists"] = lists
	for _, l := range lists {
		path := "/admin/list/" + l.Group
		for _, ls := range l.Subgroups {
			subpath := path + "/" + ls.Group
			if p.URL.Path == subpath {
				p.Data["parent_list"] = l
				l = ls
				path = subpath
				break
			}
		}
		if p.URL.Path != path &&
			!(p.URL.Path == "/admin/list" && l.Group == "all") {
			continue
		}
		p.Title = "Admin: " + l.Title
		p.Data["list"] = l
		return
	}
	p.http_error(404)
}

var account_path_rexp = regexp.MustCompile(`^/admin/account/[0-9]+$`)

func admin_account_handler(p *page) {
	if !account_path_rexp.MatchString(p.URL.Path) {
		p.http_error(404)
		return
	}
	if !p.must_be_admin() {
		return
	}
	member_id, _ := strconv.Atoi(p.URL.Path[len("/admin/account/"):])
	m := p.Get_member_by_id(member_id)
	if m == nil {
		p.http_error(404)
		return
	}
	p.Title = "Admin: @" + m.Username
	p.Data["member"] = m
	if p.PostFormValue("approve-membership") != "" {
		member_id, err := strconv.Atoi(p.PostFormValue("approve-membership"))
		if err != nil || member_id != m.Id {
			p.http_error(400)
			return
		}
		p.Member.Approve_membership(m)
		if m.Talk_user() != nil && p.PostFormValue("notify-member") == "on" {
			//TODO
		}
		p.redirect = p.URL.Path
	} else if p.PostFormValue("decline-membership") != "" {
		member_id, err := strconv.Atoi(p.PostFormValue("decline-membership"))
		if err != nil || member_id != m.Id {
			p.http_error(400)
			return
		}
		//TODO p.Decline_membership
		if m.Talk_user() != nil && p.PostFormValue("notify-member") == "on" {
			p.Talk.Message_user("Your membership request was declined",
				"Your membership request was declined by @"+p.Member.Username+
					".", m.Talk_user(), p.Member.Talk_user())
		}
	} else if p.PostFormValue("approve-free-membership") != "" {
		p.Member.Approve_free_membership(m)
		p.redirect = p.URL.Path
	} else if sub_id := p.PostFormValue("cancel-membership"); sub_id != "" {
		membership := m.Get_membership()
		if sub_id != membership.ID {
			p.http_error(400)
			return
		}
		m.Cancel_membership()
		if m.Talk_user() != nil && p.PostFormValue("notify-member") == "on" {
			p.Talk.Message_user("Your membership has been cancelled",
				"Your membership was cancelled by @"+p.Member.Username+
					".", m.Talk_user(), p.Member.Talk_user())
		}
		p.redirect = p.URL.Path
	} else if _, ok := p.PostForm["force-password-reset"]; ok {
		p.Force_password_reset(p.Config.Url(), m)
		p.Data["reset_success"] = "Reset e-mail sent"
	} else if _, ok := p.PostForm["update-registered"]; ok {
		if date, err := time.ParseInLocation("2006-01-02",
			p.PostFormValue("registered"), time.Local); err != nil {
			p.http_error(400)
			return
		} else if date.After(time.Now()) {
			p.Data["registered_error"] = "Registration date cannot be in the" +
				" future"
		} else {
			m.Set_registration_date(date)
		}
	} else if _, ok := p.PostForm["update-username"]; ok {
		p.Data["username_error"] = m.Update_username(p.PostFormValue("username"))
	} else if _, ok := p.PostForm["update-name"]; ok {
		p.Data["name_error"] = m.Update_name(p.PostFormValue("name"))
	} else if _, ok := p.PostForm["update-access-card"]; ok {
		p.Data["access_card_error"] = m.Set_key_card(p.PostFormValue("access-card"))
	} else if _, ok := p.PostForm["update-telephone"]; ok {
		p.Data["telephone_error"] = m.Set_telephone(p.PostFormValue("telephone"))
	} else if _, ok := p.PostForm["update-open-house"]; ok {
		if p.PostFormValue("open-house") == "" {
			p.Data["open_house_error"] = "Invalid date"
		} else {
			date, err := time.ParseInLocation("2006-01-02",
				p.PostFormValue("open-house"), time.Local)
			if err != nil {
				p.http_error(400)
				return
			}
			p.Data["open_house_error"] = m.Set_open_house_date(date)
		}
	} else if _, ok := p.PostForm["update-vehicle"]; ok {
		p.Data["vehicle_error"] = m.Set_vehicle(p.PostFormValue("vehicle"))
	} else if _, ok := p.PostForm["update-plate"]; ok {
		p.Data["plate_error"] = m.Set_license_plate(p.PostFormValue("plate"))
	} else if _, ok := p.PostForm["delete"]; ok {
		if _, ok = p.PostForm["access-card"]; ok {
			m.Delete_access_card()
		} else if _, ok = p.PostForm["telephone"]; ok {
			m.Delete_telephone()
		} else if _, ok = p.PostForm["open-house"]; ok {
			m.Delete_open_house_date()
		} else if _, ok = p.PostForm["vehicle"]; ok {
			m.Delete_vehicle()
		} else if _, ok = p.PostForm["plate"]; ok {
			m.Delete_license_plate()
		}
	}
}

func admin_storage_handler(p *page) {
	if !p.must_be_admin() {
		return
	}
	p.Title = "Admin: Storage"
	if p.PostFormValue("member") != "" {
		member_id, err := strconv.Atoi(p.PostFormValue("member"))
		if err != nil {
			p.http_error(400)
			return
		}
		m := p.Get_member_by_id(member_id)
		if m == nil {
			p.http_error(400)
			return
		}
		var plan_id string
		if plan_id = p.PostFormValue("approve-storage"); plan_id != "" {
			number, err := strconv.Atoi(p.PostFormValue("storage-number"))
			if err != nil {
				p.http_error(400)
				return
			}
			if m.Get_payment_source() == nil {
				p.Data["error"] = "No payment information for @" + m.Username
			} else {
				p.Data["error"] = m.New_storage_lease(plan_id, number)
			}
			p.Cancel_pending_subscription(&member.Pending_subscription{
				Member: m, Plan_id: plan_id})
		} else if plan_id = p.PostFormValue("decline-storage"); plan_id != "" {
			p.Cancel_pending_subscription(&member.Pending_subscription{
				Member: m, Plan_id: plan_id})
		} else if p.PostFormValue("cancel-storage-number") != "" {
			number, err := strconv.Atoi(p.PostFormValue("cancel-storage-number"))
			if err != nil {
				p.http_error(400)
				return
			}
			plan_id := member.Plan_category(
				p.PostFormValue("cancel-storage-plan")) + "-" +
				member.Plan_identifier(p.PostFormValue("cancel-storage-plan"))
			p.Data["error"] = m.Cancel_storage_lease(plan_id, number)
		}
	}
	pending := p.List_all_pending_subscriptions()
	for i := 0; i < len(pending); i++ {
		if !strings.HasPrefix(pending[i].Plan_id, "storage-") {
			pending = append(pending[:i], pending[i+1:]...)
			i--
		}
	}
	p.Data["storage_requests"] = pending
}

func admin_cards_handler(p *page) {
	if !p.must_be_admin() {
		return
	}
	if p.PostFormValue("update-member") != "" {
		member_id, err := strconv.Atoi(p.PostFormValue("update-member"))
		if err != nil {
			p.http_error(400)
			return
		}
		m := p.Get_member_by_id(member_id)
		if m == nil {
			p.http_error(400)
			return
		}
		if err := m.Set_key_card(p.PostFormValue("access-card")); err != nil {
			p.Data["access_card_error_member"] = m.Id
			p.Data["access_card_error"] = err
		}
	}
}

func member_upload_handler(p *page) {
	if p.Request.Method != "POST" {
		p.http_error(400)
		return
	}
	input := strings.Split(p.PostFormValue("member-upload"), "\n")
	lines := make([]string, len(input))
	copy(lines, input)
	line_error := make(map[int]string)
	line_success := make(map[int]*member.Member)
	rm_line := func(i int) {
		lines = append(lines[:i], lines[i+1:]...)
	}
line_loop:
	for i, line := range lines {
		line := strings.TrimSpace(line)
		if len(line) == 0 {
			rm_line(i)
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 3 {
			line_error[i] = "Invalid: not enough fields"
			continue
		}
		username := strings.TrimSpace(fields[0])
		name := strings.TrimSpace(fields[1])
		email := strings.TrimSpace(fields[2])
		var (
			free       bool
			key_card   string
			registered time.Time
		)
		for j, field := range fields[3:] {
			field := strings.TrimSpace(field)
			if field == "" {
				continue
			} else if field == "free" {
				free = true
			} else if member.Key_card_rexp.MatchString(field) {
				key_card = field
			} else if t, err := time.ParseInLocation("2006-01-02", field,
				time.Local); err == nil {
				registered = t
			} else {
				line_error[i] = "Field " + fmt.Sprint(j+4) +
					" is an invalid format: '" + field + "'"
				continue line_loop
			}
		}
		m, err := p.New_member(username, name, email)
		if err != nil {
			line_error[i] = err.Error()
			continue
		}
		line_success[i] = m
		rm_line(i)
		if !registered.IsZero() {
			m.Set_registration_date(registered)
		}
		if key_card != "" {
			if err := m.Set_key_card(key_card); err != nil {
				line_error[i] = err.Error()
				continue
			}
		}
		if free {
			if err := p.Member.Approve_free_membership(m); err != nil {
				line_error[i] = err.Error()
				continue
			}
		}
		p.Member.Force_password_reset(p.Config.Url(), m)
	}
	p.Data["lines"] = lines
	p.Data["line_error"] = line_error
	p.Data["line_success"] = line_success
}
