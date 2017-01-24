package site

import (
    "crypto/rand"
    "database/sql"
    "encoding/hex"
    "log"
    "net/http"
    "time"
    "golang.org/x/crypto/scrypt"
)

type Member struct {
    Username string
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

func (s *Http_server) authenticate (w http.ResponseWriter, r *http.Request) (ok bool, username string) {
    unset_cookie := func () {
        http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Expires: time.Unix(0, 0), MaxAge: -1})
    }
    cookie, err := r.Cookie("session")
    if err != nil { return }
    var (
        uname string
        expired bool
    )
    err = s.db.QueryRow("SELECT username, expired FROM session_http WHERE token = $1", cookie.Value).Scan(&uname, &expired)
    if err == sql.ErrNoRows {
        unset_cookie()
        return
    } else if err != nil {
        log.Panic(err)
    } else if expired {
        unset_cookie()
        return
    }
    new_token := rand256()
    _, err = s.db.Exec("UPDATE session_http SET token = $1, last_seen = now() WHERE token = $2", new_token, cookie.Value)
    if err != nil { log.Panic(err) }
    http.SetCookie(w, &http.Cookie{Name: "session", Value: new_token, Path: "/", Domain: Domain, /* Secure: true,*/ HttpOnly: true})
    return true, uname
}

func (s *Http_server) sign_out (w http.ResponseWriter, r *http.Request) {
    cookie, err := r.Cookie("session")
    if err == nil {
        _, err = s.db.Exec("UPDATE session_http SET expired = true WHERE token = $1", cookie.Value)
        if err != nil { log.Panic(err) }
    }
    http.SetCookie(w, &http.Cookie{Name: "session", Value: "", MaxAge: -1})
}

func (s *Http_server) sign_in () {
    s.mux.HandleFunc("/signin", func (w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/signin" {
            http.Error(w, "", http.StatusNotFound)
            return
        }
        if r.PostFormValue("signin") != "true" {
            p := page{Name: "signin", Title: "Sign in"}
            s.tmpl.Execute(w, p)
            return
        }
    })
    s.mux.HandleFunc("/signin.json", func (w http.ResponseWriter, r *http.Request) {
        username := r.PostFormValue("username")
        var (
            password_key string
            password_salt string
        )
        err := s.db.QueryRow("SELECT password_key, password_salt FROM member WHERE username = $1", username).Scan(&password_key, &password_salt)
        rsp := "success"
        if err == sql.ErrNoRows {
            rsp = "invalid username"
        } else if password_key != key(r.PostFormValue("password"), password_salt) {
            rsp = "incorrect password"
        } else {
            _, err = s.db.Exec("UPDATE session_http SET expired = true WHERE username = $1 AND expired = false", username)
            if err != nil { log.Panic(err) }
            token := rand256()
            _, err = s.db.Exec("INSERT INTO session_http (token, username) VALUES ($1, $2)", token, username)
            if err != nil { log.Panic(err) }
            cookie := &http.Cookie{Name: "session", Value: token, Path: "/", Domain: Domain, /* Secure: true,*/ HttpOnly: true}
            if r.PostFormValue("save_session") == "on" {
                cookie.Expires = time.Now().AddDate(1, 0, 0)
            }
            http.SetCookie(w, cookie)
        }
        w.Write([]byte("\"" + rsp + "\""))
    })
}
