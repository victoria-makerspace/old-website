package member

import (
	"database/sql"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/sub"
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
	plan, ok := a.Plans[p.Plan_id]
	if !ok {
		return fmt.Errorf("Invalid plan identifier '%s'", p.Plan_id)
	}
	if !p.Member.Has_card() {
		if plan.Amount != 0 {
			return fmt.Errorf("No valid payment source")
		} else if err := p.Member.Update_customer("", nil); err != nil {
			return err
		}
	}
	params := &stripe.SubParams{
		Customer: p.Member.Customer_id,
		Plan: p.Plan_id}
	params.Params.Meta = make(map[string]string)
	params.Meta["member_id"] = fmt.Sprint(p.Member.Id)
	params.Meta["approved_by"] = fmt.Sprint(a.Id)
	a.Cancel_pending_subscription(p)
	_, err := sub.New(params)
	return err
}

func (a *Member) Approve_membership(m *Member) error {
	p := m.Get_pending_membership()
	if p == nil {
		return fmt.Errorf("@%s has not requested a membership", m.Username)
	}
	if m.Get_membership() != nil {
		params := &stripe.SubParams{Plan: p.Plan_id}
		a.Cancel_pending_subscription(p)
		return m.Update_membership(params)
	}
	if m.Talk_user() != nil {
		m.talk.Add_to_group("Members")
	}
	return a.Approve_subscription(p)
}

func (a *Member) Approve_free_membership(m *Member) error {
	if m.Get_membership() != nil {
		params := &stripe.SubParams{Plan: "membership-free"}
		return m.Update_membership(params)
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

func (a *Member) Send_password_resets(members ...*Member) {
	for _, m := range members {
		token := m.create_reset_token()
		if token == "" {
			continue
		}
		msg := message{subject: "Makerspace.ca: password reset"}
		msg.set_from("Makerspace", "admin@makerspace.ca")
		msg.add_to(m.Name, m.Email)
		URL := ""//TODO: m.Config["url"].(string)
		msg.body = "Hello " + m.Name + " (@" + m.Username + "),\n\n" +
			"A password reset has been requested for your " + URL +
			" account on behalf of an administrator (@" + a.Username +
			").\n\n" +
			"Reset your password by visiting " +
			URL + "/sso/reset?token=" + token + ".\n\n" +
			"Your password-reset token will expire in " +
			m.Password_reset_window +
			", you can request a new reset token at " +
			URL + "/sso/reset?username=" + url.QueryEscape(m.Username) +
			"&email=" + url.QueryEscape(m.Email) + ".\n\n"
		m.send_email("admin@makerspace.ca", msg.emails(), a.format_message(msg))
	}
}
