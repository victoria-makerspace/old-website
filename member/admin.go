package member

import (
	"database/sql"
	"github.com/lib/pq"
	"log"
	"net/url"
)

type Admin struct {
	privileges []string
}

func (m *Member) get_admin() {
	var privileges pq.StringArray
	if err := m.QueryRow(
		"SELECT privileges "+
			"FROM administrator "+
			"WHERE member = $1", m.Id).
		Scan(&privileges); err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return
	}
	m.Admin = &Admin{privileges}
}

// Approve_member sets the approval flag on <m> and activates the invoice if
//	m.Membership_invoice exists, otherwise setting the gratuitous flag.
//BUG: approving a member with unverified e-mail will leave the member out of the "Members" talk group, requiring manual intervention
func (a *Member) Approve_member(m *Member) {
	if a.Admin == nil {
		log.Panicf("%s is not an administrator\n", a.Username)
	}
	if m.Approved {
		log.Panicf("%s is already an approved member\n", m.Username)
	}
	if _, err := m.Exec(
		"UPDATE member "+
		"SET"+
		"	approved_at = now(),"+
		"	approved_by = $1 ", a.Id); err != nil {
		log.Panic(err);
	}
	m.Approved = true
	if m.Talk_user() != nil {
		if err := m.talk.Add_to_group("Members"); err != nil {
			log.Println(err)
		}
	}
	if m.Membership_invoice != nil {
		m.Payment().Approve_pending_membership(m.Membership_invoice)
	} else {
		m.set_gratuitous(true)
	}
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
		URL := m.Config["url"].(string)
		msg.body = "Hello " + m.Name + " (@" + m.Username + "),\n\n" +
			"A password reset has been requested for your "+ URL +
			"account on behalf of an administrator (@" + a.Username +
			").\n\n"+
			"Reset your password by visiting " +
			URL + "/sso/reset?token=" + token + ".\n\n"+
			"Your password-reset token will expire in " +
			m.Config["password-reset-window"].(string) +
			", you can request a new reset token at " +
			URL + "/sso/reset?username=" + url.QueryEscape(m.Username) +
			"&email=" + url.QueryEscape(m.Email) + ".\n\n"
		m.send_email("admin@makerspace.ca", msg.emails(), msg.format())
	}
}
