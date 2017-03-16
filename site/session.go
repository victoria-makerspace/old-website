package site

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/vvanpo/makerspace/member"
	"log"
	"net/http"
	"time"
)

//TODO: maybe save sessions in a slice in the http_server object, to persist member data across requests?

func (p *page) set_session_cookie(value string, expires bool) {
	cookie := &http.Cookie{
		Name:   "session",
		Value:  value,
		Path:   "/",
		Domain: p.config["domain"].(string),
		/* Secure: true, */
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
	var expires pq.NullTime
	// Select non-expired sessions
	if err := p.db.QueryRow("SELECT member, expires FROM session_http WHERE token = $1 AND (expires > now() OR expires IS NULL)", cookie.Value).Scan(&member_id, &expires); err != nil {
		if err == sql.ErrNoRows {
			// Invalid session cookie
			p.unset_session_cookie()
			return
		}
		log.Panic(err)
	}
	p.Session = &Session{Member: p.Get_member_by_id(member_id),
		token: member.Rand256()}
	if !p.Session.Member.Verified_email() {
		log.Panic("Invalid session found from unverified member.")
	}
	p.set_session_cookie(p.Session.token, expires.Valid)
	if _, err := p.db.Exec("UPDATE session_http SET token = $1, last_seen = now(), expires = now() + interval '1 year' WHERE token = $2", p.Session.token, cookie.Value); err != nil {
		log.Panic(err)
	}
}

// destroy_session invalidates the current session.
func (p *page) destroy_session() {
	if p.Session == nil {
		return
	}
	if _, err := p.db.Exec("UPDATE session_http SET expires = 'epoch' WHERE token = $1", p.Session.token); err != nil {
		log.Panic(err)
	}
	p.unset_session_cookie()
}

/*
//TODO: refactor this mess
// talk_user_data fetches user info from the talk server
func (p *page) talk_user_data() {
	var data map[string]interface{}
	rsp, err := http.Get(p.Talk_url + "/users/" + p.Member().Username + ".json")
	if err != nil {
		log.Println(err)
		return
	}
	if err = json.NewDecoder(rsp.Body).Decode(&data); err != nil {
		log.Println(err)
		return
	}
	if user, ok := data["user"].(map[string]interface{}); ok {
		p.Field["avatar_url"] = p.Talk_url +
			string(avatar_size_rexp.ReplaceAll([]byte(user["avatar_template"].(string)), []byte("120")))
		p.Field["card_background_url"] = p.Talk_url + user["card_background"].(string)
		p.Field["profile_background_url"] = p.Talk_url + user["profile_background"].(string)
	} else {
		log.Printf("Error requesting talk user data for '%s': %q\n", p.Member().Username, data)
		return
	}
	// Get notifications
	data = nil
	rsp, err = http.Get(p.Talk_url + "/notifications.json?api_key=" + p.config.Discourse["api-key"] + "&api_username=" + p.Member().Username)
	if err != nil {
		log.Println(err)
		return
	}
	if err = json.NewDecoder(rsp.Body).Decode(&data); err != nil {
		log.Println(err)
		return
	}
	if _, ok := data["notifications"].([]interface{}); !ok {
		log.Printf("Error requesting talk notification data for '%s': %q\n", p.Member().Username, data)
		return
	}
	ntfns := make([]struct {
		Notification_type int
		Notification_icon string
		Read              bool
		Created_at        string
		Post_number       int
		Topic_id          int
		Slug              string
		Data              map[string]interface{}
	}, 12)
	len := 0
	for i, v := range data["notifications"].([]interface{})[:12] {
		if n, ok := v.(map[string]interface{}); ok {
			len++
			ntfns[i].Notification_type = int(n["notification_type"].(float64))
			ntfns[i].Read = n["read"].(bool)
			ntfns[i].Created_at = n["created_at"].(string)
			if pn, ok := n["post_number"].(float64); ok {
				ntfns[i].Post_number = int(pn)
			}
			if ti, ok := n["topic_id"].(float64); ok {
				ntfns[i].Topic_id = int(ti)
			}
			if sl, ok := n["slug"].(string); ok {
				ntfns[i].Slug = sl
			}
			ntfns[i].Data = n["data"].(map[string]interface{})
			var icon string
			switch ntfns[i].Notification_type {
			case 2:
				icon = "undo"
			case 3:
			case 4:
			case 5:
			case 6:
				icon = "envelope"
			case 7:
			case 8:
			case 9:
			case 10:
			case 11:
			case 12:
				icon = "certificate"
			case 16:
				icon = "group"
			}
			ntfns[i].Notification_icon = icon
		}
	}
	p.Field["notifications"] = ntfns[:len]
}*/
