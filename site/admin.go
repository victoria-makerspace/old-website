package site

import (
	"fmt"
	"github.com/vvanpo/makerspace/member"
	"strconv"
	"strings"
	"time"
	"regexp"
)

func init() {
	init_handler("admin", admin_handler, "/admin")
	init_handler("admin-manage", manage_account_handler, "/admin/account/")
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
	if p.PostFormValue("approve-membership") != "" {
		member_id, err := strconv.Atoi(p.PostFormValue("approve-membership"))
		if err != nil {
			p.http_error(400)
			return
		}
		if member := p.Get_member_by_id(member_id); member == nil {
			p.http_error(400)
		} else if !member.Approved {
			p.Member.Approve_member(member)
			p.Data["Member_approved"] = member
		} else {
			p.http_error(500)
		}
		return
	} else if p.PostFormValue("decline-membership") != "" {
		member_id, err := strconv.Atoi(p.PostFormValue("decline-membership"))
		if err != nil {
			p.http_error(400)
			return
		}
		member := p.Get_member_by_id(member_id)
		if member == nil {
			p.http_error(400)
			return
		}
		if member.Talk_user() != nil {
			p.Message_member("Your membership was declined",
				"Your membership request was declined by @"+p.Member.Username+
				".", member.Talk_user(), p.Member.Talk_user())
		}
		member.Cancel_membership()
	} else if p.PostFormValue("member-upload") != "" {
		member_upload_handler(p)
	}
}

var account_path_rexp = regexp.MustCompile(`^/admin/account/[0-9]+$`)

func manage_account_handler(p *page) {
	if !p.must_be_admin() {
		return
	}
	if !account_path_rexp.MatchString(p.URL.Path) {
		p.http_error(404)
		return
	}
	member_id, _ := strconv.Atoi(p.URL.Path[len("/admin/account/"):])
	m := p.Get_member_by_id(member_id)
	if m == nil {
		p.http_error(404)
		return
	}
	p.Title = "Admin panel - @" + m.Username
	p.Data["member"] = m
	if p.PostFormValue("approve-membership") != "" {
		member_id, err := strconv.Atoi(p.PostFormValue("approve-membership"))
		if err != nil || member_id != m.Id {
			p.http_error(400)
		} else if !m.Approved {
			p.Member.Approve_member(m)
		} else {
			p.http_error(500)
		}
	} else if p.PostFormValue("decline-membership") != "" {
		member_id, err := strconv.Atoi(p.PostFormValue("decline-membership"))
		if err != nil || member_id != m.Id {
			p.http_error(400)
			return
		}
		if m.Talk_user() != nil {
			p.Message_member("Your membership was declined",
				"Your membership request was declined by @"+p.Member.Username+
				".", m.Talk_user(), p.Member.Talk_user())
		}
		m.Cancel_membership()
	} else if _, ok := p.PostForm["terminate_membership"]; ok {
		if m.Talk_user() != nil {
			p.Message_member("Your membership has been cancelled",
				"Your membership request was cancelled by @"+p.Member.Username+
				".", m.Talk_user(), p.Member.Talk_user())
		}
		m.Cancel_membership()
	} else if _, ok := p.PostForm["terminate"]; ok && m.Payment() != nil {
		id, _ := strconv.Atoi(p.PostFormValue("terminate"))
		if invoice := m.Payment().Get_bill(id); invoice != nil {
			m.Payment().Cancel_recurring_bill(invoice)
		}
	} else if p.PostFormValue("registered") != "" {
		if registered, err := time.ParseInLocation("2006-01-02",
			p.PostFormValue("registered"), time.Local); err != nil {
			p.http_error(400)
		} else if registered.After(time.Now()) {
			p.Data["registered_error"] = "Invalid input date"
		} else {
			m.Set_registration_date(registered)
		}
	} else if name := p.PostFormValue("name"); name != "" {
		if err := m.Set_name(name); err != nil {
			p.Data["name_error"] = err
		}
	} else if p.PostFormValue("key-card") != "" {
		if err := m.Set_key_card(p.PostFormValue("key-card")); err != nil {
			p.Data["key_card_error"] = err
		}
	} else if tel := p.PostFormValue("telephone"); tel != "" {
		if err := m.Set_telephone(tel); err != nil {
			p.Data["telephone_error"] = err
		}
	} else if p.PostFormValue("update-type") == fmt.Sprint(m.Id) {
		if m.Admin == nil && p.PostFormValue("type") == "admin" {
			//TODO
		}
	} else if p.PostFormValue("delete-account") == fmt.Sprint(m.Id) {
		//TODO
	}
}

func member_upload_handler(p *page) {
	if p.Request.Method != "POST" {
		p.http_error(400)
		return
	}
	type new_member struct {
		line                  int
		username, name, email string
		date                  time.Time
		free                  bool
		verified              bool
		key_card			  string
	}
	new_members := make([]new_member, 0)
	lines := strings.Split(p.PostFormValue("member-upload"), "\n")
	line_error := make([][]string, len(lines))
	line_success := make([]*member.Member, len(lines))
	for i, line := range lines {
		line := strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 3 {
			line_error[i] = []string{"Invalid: not enough fields"}
			continue
		}
		nm := new_member{
			line:     i,
			username: strings.TrimSpace(fields[0]),
			name:     strings.TrimSpace(fields[1]),
			email:    strings.TrimSpace(fields[2])}
		for j, field := range fields[3:] {
			field := strings.TrimSpace(field)
			if field == "" {
				continue
			} else if field == "free" {
				nm.free = true
			} else if field == "verified" {
				nm.verified = true
			} else if member.Key_card_rexp.MatchString(field) {
				nm.key_card = field
			} else if t, err := time.ParseInLocation("2006-01-02", field,
				time.Local); err == nil {
				nm.date = t
			} else {
				line_error[i] = []string{"Field " + fmt.Sprint(j + 4) +
					" invalid: '" + field + "'"}
				break
			}
		}
		new_members = append(new_members, nm)
	}
	verified := make([]*member.Member, 0)
	for _, nm := range new_members {
		m, err := p.New_member(nm.username, nm.email, nm.name)
		if m == nil {
			line_error[nm.line] = make([]string, 0)
			for _, v := range err {
				line_error[nm.line] = append(line_error[nm.line], v)
			}
			continue
		}
		if !nm.date.IsZero() {
			m.Set_registration_date(nm.date)
		}
		if nm.verified {
			if err := m.Verify_email(nm.email); err != nil {
				line_error[nm.line] = []string{"E-mail verification failed"}
			} else {
				verified = append(verified, m)
			}
		} else {
			m.Send_email_verification(nm.email)
		}
		if nm.free {
			p.Member.Approve_member(m)
		}
		if nm.key_card != "" {
			if err := m.Set_key_card(nm.key_card); err != nil {
				e := []string{err.Error()}
				if line_error[nm.line] == nil {
					line_error[nm.line] = e
				} else {
					line_error[nm.line] = append(line_error[nm.line], e...)
				}
			}
		}
		line_success[nm.line] = m
		lines[nm.line] = ""
	}
	p.Data["lines"] = lines
	p.Data["line_error"] = line_error
	p.Data["line_success"] = line_success
	p.Member.Send_password_resets(verified...)
}
