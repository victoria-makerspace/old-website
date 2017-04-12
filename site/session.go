package site

import (
	"database/sql"
	"github.com/vvanpo/makerspace/member"
	"log"
	"net/http"
	"time"
)

//TODO: maybe save sessions in a slice in the http_server object, to persist member data across requests?

func (p *page) set_session_cookie(value string, expires bool) {
	cookie := &http.Cookie{
		Name:     "session",
		Value:    value,
		Path:     "/",
		Domain:   p.config["domain"].(string),
		Secure:   p.config["tls"].(bool),
		HttpOnly: true}
	// If not set to expire, set expiry date for a year from now.
	if !expires {
		cookie.Expires = time.Now().AddDate(1, 0, 0)
	}
	p.cookies["session"] = cookie
}

func (p *page) unset_session_cookie() {
	p.cookies["session"] = &http.Cookie{
		Name:     "session",
		Value:    " ",
		Path:     "/",
		Domain:   p.config["domain"].(string),
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true}
}

type Session struct {
	*member.Member
	token string
}

// new_session fails silently on unverified accounts
func (p *page) new_session(m *member.Member, expires bool) {
	if !m.Verified_email() {
		return
	}
	token := member.Rand256()
	query := "INSERT INTO session_http (token, member, expires) VALUES ($1, $2, "
	// TODO: purge null expiries from database occasionally, since browsers don't
	//	open forever..
	if expires {
		query += "null)"
	} else {
		query += "now() + interval '1 year')"
	}
	if _, err := p.db.Exec(query, token, m.Id); err != nil {
		log.Panic(err)
	}
	p.set_session_cookie(token, expires)
	p.Session = &Session{Member: m, token: token}
}

// authenticate validates the session cookie, setting p.Session if valid
func (p *page) authenticate() {
	cookie, err := p.Cookie("session")
	// If there is no session cookie, or session already exists, return
	if err != nil || p.Session != nil {
		return
	}
	var member_id int
	// Select non-expired sessions
	if err := p.db.QueryRow(
		"SELECT member "+
		"FROM session_http "+
		"WHERE token = $1"+
		"	AND (expires > now() OR expires IS NULL)",
		cookie.Value).Scan(&member_id); err != nil {
		if err == sql.ErrNoRows {
			// Invalid session cookie
			p.unset_session_cookie()
			return
		}
		log.Panic(err)
	}
	p.Session = &Session{
		Member: p.Get_member_by_id(member_id),
		token: cookie.Value}
	if !p.Session.Member.Verified_email() {
		log.Panic("Invalid session found from unverified member.")
	}
	if _, err := p.db.Exec(
		"UPDATE session_http "+
		"SET last_seen = now() "+
		"WHERE token = $1", p.Session.token); err != nil {
		log.Panic(err)
	}
}

// destroy_session invalidates the current session.
func (p *page) destroy_session() {
	if p.Session == nil {
		return
	}
	if _, err := p.db.Exec(
		"UPDATE session_http "+
		"SET expires = 'epoch' "+
		"WHERE token = $1", p.Session.token); err != nil {
		log.Panic(err)
	}
	p.unset_session_cookie()
}
