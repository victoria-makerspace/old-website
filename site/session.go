package site

import (
	"database/sql"
	"encoding/json"
	"github.com/lib/pq"
	"github.com/vvanpo/makerspace/member"
	"log"
	"net/http"
	"regexp"
	"time"
)

func (p *page) set_session_cookie(value string, expires bool) {
	cookie := &http.Cookie{Name: "session",
		Value:  value,
		Path:   "/",
		Domain: p.config.Domain,
		/* Secure: true, */
		HttpOnly: true}
	// If not set to expire, set expiry date for a year from now.
	if !expires {
		cookie.Expires = time.Now().AddDate(1, 0, 0)
	}
	http.SetCookie(p.ResponseWriter, cookie)
}

func (p *page) unset_session_cookie() {
	cookie := &http.Cookie{Name: "session",
		Value:   " ",
		Path:    "/",
		Domain:  p.config.Domain,
		Expires: time.Unix(0, 0),
		MaxAge:  -1,
		/* Secure: true, */
		HttpOnly: true}
	p.ResponseWriter.Header().Set("Set-Cookie", cookie.String())
}

type session struct {
	*member.Member
	token   string
}

// new_session
func (p *page) new_session(m *member.Member, expires bool) {
	token := member.Rand256()
	query := "INSERT INTO session_http (token, username, expires) VALUES ($1, $2, "
	// TODO: purge null expiries from database occasionally, since browsers don't
	//	open forever..
	if expires {
		query += "null)"
	} else {
		query += "now() + interval '1 year')"
	}
	if _, err := p.db.Exec(query, token, m.Username); err != nil {
		log.Panic(err)
	}
	p.set_session_cookie(token, expires)
	p.Session = &session{Member: m, token: token}
	p.talk_user_data()
}

// authenticate validates the session cookie, setting p.Session if valid
func (p *page) authenticate() {
	cookie, err := p.Cookie("session")
	// If there is no session cookie, or session already exists, return
	if err != nil || p.Session != nil {
		return
	}
	var username string
	var expires pq.NullTime
	// Select non-expired sessions
	if err := p.db.QueryRow("SELECT username, expires FROM session_http WHERE token = $1 AND (expires > now() OR expires IS NULL)", cookie.Value).Scan(&username, &expires); err != nil {
		if err == sql.ErrNoRows {
			// Invalid session cookie
			p.unset_session_cookie()
			return
		}
		log.Panic(err)
	}
	p.Session = &session{Member: member.Get(username, p.db),
		token: member.Rand256()}
	p.set_session_cookie(p.Session.token, expires.Valid)
	if _, err := p.db.Exec("UPDATE session_http SET token = $1, last_seen = now(), expires = now() + interval '1 year' WHERE token = $2", p.Session.token, cookie.Value); err != nil {
		log.Panic(err)
	}
	p.talk_user_data()
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

var avatar_size_rexp = regexp.MustCompile("{size}")

// talk_user_data fetches user info from the talk server
func (p *page) talk_user_data() {
	var data map[string]interface{}
	talk_url := p.config.Discourse["url"]
	rsp, err := http.Get(talk_url + "/users/" + p.Member().Username + ".json")
	if err != nil || json.NewDecoder(rsp.Body).Decode(&data) != nil {
		log.Println(err)
		return
	}
	if user, ok := data["user"].(map[string]interface{}); ok {
		p.Field["avatar_url"] = talk_url + string(avatar_size_rexp.ReplaceAll([]byte(user["avatar_template"].(string)), []byte("120")))
		p.Field["card_background_url"] = talk_url + user["card_background"].(string)
		p.Field["profile_background_url"] = talk_url + user["profile_background"].(string)
	}
	// Get notifications
	data = nil
	rsp, err = http.Get(talk_url + "/notifications.json?api_key=" + p.config.Discourse["api-key"] + "&api_username=" + p.Member().Username)
	if err != nil || json.NewDecoder(rsp.Body).Decode(&data) != nil {
		log.Println(err)
		return
	}
ntfns := make([]struct{
		Notification_type int
		Notification_icon string
		Read bool
		Created_at string
		Post_number int
		Topic_id int
		Slug string
		Data map[string]interface{}}, 16)
	for i, v := range data["notifications"].([]interface{})[:12] {
		if n, ok := v.(map[string]interface{}); ok {
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
			case 2: icon = "undo"
			case 3:
			case 4:
			case 5:
			case 6: icon = "envelope"
			case 7:
			case 8:
			case 9:
			case 10:
			case 11:
			case 12: icon = "certificate"
			}
			ntfns[i].Notification_icon = icon
		}
	}
	p.Field["notifications"] = ntfns
}
