package site

import (
    "crypto/rand"
    "database/sql"
    "encoding/hex"
    "log"
    "net/http"
    "time"
    "golang.org/x/crypto/scrypt"
    "github.com/lib/pq"
)

type Member struct {
    Session string
    Username string
    Name string
}

func rand256 () string {
    n := make([]byte, 32)
    _, err := rand.Read(n)
    if err != nil { log.Panic(err) }
    return hex.EncodeToString(n)
}

func key (password, salt string) string {
    s, err := hex.DecodeString(salt)
    if err != nil { log.Panicf("Invalid salt: %s", err) }
    key, err := scrypt.Key([]byte(password), s, 16384, 8, 1, 32);
    if err != nil { log.Panic(err) }
    return hex.EncodeToString(key)
}

func (s *Http_server) authenticate (w http.ResponseWriter, r *http.Request, member *Member) {
    cookie, err := r.Cookie("session")
    if err != nil { return }
    var (
        uname string
        expires pq.NullTime
    )
    err = s.db.QueryRow("SELECT username, expires FROM session_http WHERE token = $1", cookie.Value).Scan(&uname, &expires)
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
    rsp_cookie := http.Cookie{Name: "session", Value: new_token, Path: "/", Domain: s.config.Domain, /* Secure: true,*/ HttpOnly: true}
    if expires.Valid {
        expires_unix := time.Now().AddDate(1, 0, 0)
        rsp_cookie.Expires = expires_unix
        _, err = s.db.Exec("UPDATE session_http SET token = $1, last_seen = now(), expires = $2 WHERE token = $3", new_token, expires_unix, cookie.Value)
    } else {
        _, err = s.db.Exec("UPDATE session_http SET token = $1, last_seen = now() WHERE token = $2", new_token, cookie.Value)
    }
    if err != nil { log.Panic(err) }
    member.Username = uname
    member.Session = new_token
    http.SetCookie(w, &rsp_cookie)
}

func (s *Http_server) sign_in (w http.ResponseWriter, r *http.Request) (username, password bool) {
    uname := r.PostFormValue("username")
    var (
        password_key string
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
    if err != nil { log.Panic(err) }
    cookie := &http.Cookie{Name: "session", Value: token, Path: "/", Domain: s.config.Domain, /* Secure: true,*/ HttpOnly: true}
    if r.PostFormValue("save_session") == "on" {
        cookie.Expires = time.Now().AddDate(1, 0, 0)
        _, err = s.db.Exec("UPDATE session_http SET expires = $1 WHERE token = $2", cookie.Expires, token)
        if err != nil { log.Panic(err) }
    }
    http.SetCookie(w, cookie)
    return true, true
}

func (s *Http_server) sign_out (w http.ResponseWriter, member *Member) {
    member.Username = ""
    if member.Session != "" {
        _, err := s.db.Exec("UPDATE session_http SET expires = 'epoch' WHERE token = $1", member.Session)
        if err != nil { log.Panic(err) }
    }
    member.Session = ""
    w.Header().Del("Set-Cookie")
    http.SetCookie(w, &http.Cookie{Name: "session", Value: " ", Path: "/", Domain: s.config.Domain, Expires: time.Unix(0, 0), MaxAge: -1, /* Secure: true,*/ HttpOnly: true})
}

func (s *Http_server) dashboard_handler () {
    s.mux.HandleFunc("/member", func (w http.ResponseWriter, r *http.Request) {
//////
s.parse_templates()
/////
        p := page{Name: "dashboard", Title: "Dashboard"}
        if r.PostFormValue("sign-in") == "true" {
            if username, password := s.sign_in(w, r); username && password {
            }
        }
        s.authenticate(w, r, &p.Member)
        if !p.Authenticated() {
            if r.PostFormValue("sign-in") != "true" {
                p := page{Name: "sign-in", Title: "Sign in"}
                s.tmpl.Execute(w, p)
                return
            } else {
                http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
                return
            }
        }
        s.tmpl.Execute(w, p)
    })
    s.mux.HandleFunc("/sign-in.json", func (w http.ResponseWriter, r *http.Request) {
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

func (s *Http_server) tools_handler () {
    s.mux.HandleFunc("/member/tools", func (w http.ResponseWriter, r *http.Request) {
        p := page{Name: "tools", Title: "Tools"}
        s.tmpl.Execute(w, p)
    })
}
