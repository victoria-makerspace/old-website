package site

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/lib/pq"
	"golang.org/x/crypto/scrypt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

type member struct {
	Session   string
	Username  string
	Name      string
	Email     string
	Talk_user struct {
		User struct {
			Id                 int
			Username           string
			Avatar_template    string
			Admin              bool
			Profile_background string
			Card_background    string
		}
	}
	Billing billing
}

func (m member) Authenticated() bool {
	if m.Session == "" {
		return false
	}
	return true
}

func (m member) Avatar() string {
	rexp := regexp.MustCompile("{size}")
	if m.Talk_user.User.Avatar_template == "" {
		return ""
	}
	return "/talk" + string(rexp.ReplaceAll([]byte(m.Talk_user.User.Avatar_template), []byte("120")))
}

func rand256() string {
	n := make([]byte, 32)
	_, err := rand.Read(n)
	if err != nil {
		log.Panic(err)
	}
	return hex.EncodeToString(n)
}

func key(password, salt string) string {
	s, err := hex.DecodeString(salt)
	if err != nil {
		log.Panicf("Invalid salt: %s", err)
	}
	key, err := scrypt.Key([]byte(password), s, 16384, 8, 1, 32)
	if err != nil {
		log.Panic(err)
	}
	return hex.EncodeToString(key)
}

func (s *Http_server) authenticate(w http.ResponseWriter, r *http.Request, member *member) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return
	}
	var (
		uname   string
		name    string
		email   string
		expires pq.NullTime
	)
	err = s.db.QueryRow("SELECT m.username, m.name, m.email, s.expires FROM session_http s INNER JOIN member m ON s.username = m.username WHERE s.token = $1", cookie.Value).Scan(&uname, &name, &email, &expires)
	if err == sql.ErrNoRows {
		s.sign_out(w, member)
		return
	} else if err != nil {
		log.Panic(err)
	} else if expires.Valid && expires.Time.Before(time.Now()) {
		s.sign_out(w, member)
		return
	}
	new_token := rand256()
	rsp_cookie := http.Cookie{Name: "session", Value: new_token, Path: "/", Domain: s.config.Domain /* Secure: true,*/, HttpOnly: true}
	if expires.Valid {
		expires_unix := time.Now().AddDate(1, 0, 0)
		rsp_cookie.Expires = expires_unix
		_, err = s.db.Exec("UPDATE session_http SET token = $1, last_seen = now(), expires = $2 WHERE token = $3", new_token, expires_unix, cookie.Value)
	} else {
		_, err = s.db.Exec("UPDATE session_http SET token = $1, last_seen = now() WHERE token = $2", new_token, cookie.Value)
	}
	if err != nil {
		log.Panic(err)
	}
	////////////
	rsp_talk, err := http.Get("http://localhost:1080/talk/users/" + uname + ".json")
	/////
	if err != nil {
		log.Panic(err)
	}
	j := json.NewDecoder(rsp_talk.Body)
	err = j.Decode(&member.Talk_user)
	if err != nil {
		log.Println("Unresponsive Talk server: " + err.Error())
	}
	member.Session = new_token
	member.Username = uname
	member.Name = name
	member.Email = email
	http.SetCookie(w, &rsp_cookie)
}

func (s *Http_server) sign_in(w http.ResponseWriter, r *http.Request) (username, password bool) {
	uname := r.PostFormValue("username")
	var (
		password_key  string
		password_salt string
	)
	err := s.db.QueryRow("SELECT password_key, password_salt FROM member WHERE username = $1", uname).Scan(&password_key, &password_salt)
	if err == sql.ErrNoRows {
		return false, false
	} else if password_key != key(r.PostFormValue("password"), password_salt) {
		return true, false
	}
	token := rand256()
	_, err = s.db.Exec("INSERT INTO session_http (token, username) VALUES ($1, $2)", token, uname)
	if err != nil {
		log.Panic(err)
	}
	cookie := &http.Cookie{Name: "session", Value: token, Path: "/", Domain: s.config.Domain /* Secure: true,*/, HttpOnly: true}
	if r.PostFormValue("save_session") == "on" {
		cookie.Expires = time.Now().AddDate(1, 0, 0)
		_, err = s.db.Exec("UPDATE session_http SET expires = $1 WHERE token = $2", cookie.Expires, token)
		if err != nil {
			log.Panic(err)
		}
	}
	http.SetCookie(w, cookie)
	return true, true
}

func (s *Http_server) sign_out(w http.ResponseWriter, m *member) {
	if m.Authenticated() {
		_, err := s.db.Exec("UPDATE session_http SET expires = 'epoch' WHERE token = $1", m.Session)
		if err != nil {
			log.Panic(err)
		}
	}
	m.Session = ""
	w.Header().Del("Set-Cookie")
	http.SetCookie(w, &http.Cookie{Name: "session", Value: " ", Path: "/", Domain: s.config.Domain, Expires: time.Unix(0, 0), MaxAge: -1 /* Secure: true,*/, HttpOnly: true})
}

func (s *Http_server) join(username, name, email, password string) bool {
	salt := rand256()
	password_key := key(password, salt)
	result, _ := s.db.Exec("INSERT INTO member (username, name, password_key, password_salt, email) VALUES ($1, $2, $3, $4, $5)", username, name, password_key, salt, email)
	n, err := result.RowsAffected()
	if n == 0 {
		return false
	} else if err != nil {
		log.Panic(err)
	}
	return true
}

func (s *Http_server) sso_handler() {
	s.mux.HandleFunc("/sso", func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query()
		if v.Get("sso") == "" {
			http.Redirect(w, r, "/talk/login", 303)
			return
		}
		payload, err := base64.StdEncoding.DecodeString(v.Get("sso"))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		sig, err := hex.DecodeString(v.Get("sig"))
		if err != nil {
			http.Error(w, http.StatusText(400), 400)
			return
		}
		mac := hmac.New(sha256.New, []byte(s.config.Discourse["sso-secret"]))
		mac.Write([]byte(v.Get("sso")))
		if !hmac.Equal(mac.Sum(nil), sig) {
			http.Error(w, http.StatusText(400), 400)
			return
		}
		var m member
		s.authenticate(w, r, &m)
		if !m.Authenticated() {
			w.WriteHeader(401)
			p := page{Name: "sign-in", Title: "Sign in"}
			s.tmpl.Execute(w, p)
			return
		}
		q, err := url.ParseQuery(string(payload))
		if err != nil {
			http.Error(w, http.StatusText(400), 400)
			return
		}
		query := url.Values{}
		query.Set("nonce", q.Get("nonce"))
		query.Set("email", m.Email)
		query.Set("username", m.Username)
		query.Set("require_activation", "true")
		query.Set("external_id", m.Username)
		p := base64.StdEncoding.EncodeToString([]byte(query.Encode()))
		mac.Reset()
		mac.Write([]byte(p))
		s := hex.EncodeToString(mac.Sum(nil))
		http.Redirect(w, r, q.Get("return_sso_url")+"?sso="+url.QueryEscape(p)+"&sig="+s, 303)
	})
}

func (s *Http_server) dashboard_handler() {
	s.mux.HandleFunc("/member", func(w http.ResponseWriter, r *http.Request) {
		//////
		s.parse_templates()
		/////
		p := page{Name: "dashboard", Title: "Dashboard"}
		s.authenticate(w, r, &p.Member)
		if !p.Member.Authenticated() {
			if r.PostFormValue("sign-in") == "true" {
				if username, password := s.sign_in(w, r); username && password {
				}
			} else {
				p := page{Name: "sign-in", Title: "Sign in"}
				s.tmpl.Execute(w, p)
				return
			}
		}
		s.tmpl.Execute(w, p)
	})
	s.mux.HandleFunc("/sign-in.json", func(w http.ResponseWriter, r *http.Request) {
		username, password := s.sign_in(w, r)
		var rsp string
		if username && password {
			rsp = "success"
		} else if username {
			rsp = "incorrect password"
		} else {
			rsp = "invalid username"
		}
		w.Write([]byte("\"" + rsp + "\""))
	})
}

func (s *Http_server) tools_handler() {
	s.mux.HandleFunc("/member/tools", func(w http.ResponseWriter, r *http.Request) {
		p := page{Name: "tools", Title: "Tools"}
		if !p.Member.Authenticated() {
			http.Error(w, http.StatusText(403), 403)
			return
		}
		s.tmpl.Execute(w, p)
	})
}
