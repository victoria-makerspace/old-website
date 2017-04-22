package member

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"time"
)

type Admin struct {
	privileges []string
}

func (ms *Members) Get_all_pending_subscriptions() []*Pending_subscription {
	pending := make([]*Pending_subscription, 0)
	rows, err := ms.Query(
		"SELECT member, requested_at, plan_id " +
		"FROM pending_subscription " +
		"ORDER BY requested_at DESC")
	defer rows.Close()
	if err != nil && err != sql.ErrNoRows {
		log.Panic(err)
	}
	for rows.Next() {
		var p Pending_subscription
		var member_id int
		if err = rows.Scan(&member_id, &p.Requested_at, &p.Plan_id);
			err != nil {
			log.Panic(err)
		}
		p.Member = ms.Get_member_by_id(member_id)
		pending = append(pending, &p)
	}
	return pending
}

func (a *Member) Approve_subscription(p *Pending_subscription) error {
	//TODO: notify member if below subscription fails
	a.Cancel_pending_subscription(p)
	//TODO: plan quantity
	return p.Member.New_subscription_item(p.Plan_id, 1)
}

func (a *Member) Approve_membership(m *Member) error {
	p := m.Get_pending_membership()
	if p == nil {
		return fmt.Errorf("@%s has not requested a membership", m.Username)
	}
	if m.Get_membership() != nil {
		a.Cancel_pending_subscription(p)
		return m.Update_membership(p.Plan_id)
	}
	if m.Talk_user() != nil {
		m.talk.Add_to_group("Members")
	}
	return a.Approve_subscription(p)
}

func (a *Member) Approve_free_membership(m *Member) error {
	if m.Get_membership() != nil {
		return m.Update_membership("membership-free")
	}
	if p := m.Get_pending_membership(); p != nil {
		a.Cancel_pending_subscription(p)
	}
	p := &Pending_subscription{m, time.Now(), "membership-free"}
	if m.Talk_user() != nil {
		m.talk.Add_to_group("Members")
	}
	return a.Approve_subscription(p)
}

func (m *Member) Clear_password() {
	if _, err := m.Exec(
		"UPDATE member "+
		"SET password_key = NULL,"+
		"	password_salt = NULL "+
		"WHERE id = $1", m.Id); err != nil {
		log.Panic(err)
	}
}

//TODO: this is messy, passing config values from the site package.  Must be a
//	cleaner way of doing this... probably by passing back a template (should
//	then be built into send_email())
func (a *Member) Force_password_reset(domain string, m *Member) {
	m.Clear_password()
	token := m.create_reset_token()
	msg := message{subject: "Password reset"}
	msg.set_from("Makerspace", "admin@makerspace.ca")
	msg.add_to(m.Name, m.Email)
	msg.body = "Hello " + m.Name + " (@" + m.Username + "),\n\n" +
		"A password reset has been requested for your Makerspace " +
		" account on behalf of an administrator (@" + a.Username +
		").\n\n" +
		"Reset your password by visiting " +
		domain + "/sso/reset?token=" + token + " to gain access to your " +
		"account.\n\nYour password-reset token will expire in " +
		m.Password_reset_window +
		", you can request a new reset token at " +
		domain + "/sso/reset?username=" + url.QueryEscape(m.Username) +
		"&email=" + url.QueryEscape(m.Email) + ".\n\n"
	m.send_email("admin@makerspace.ca", msg.emails(), a.format_message(msg))
}
