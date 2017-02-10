package site

import (
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"net/http"
	"crypto/sha256"
	"crypto/hmac"
	"github.com/vvanpo/makerspace/member"
)

func (s *Http_server) parse_sso_req(w http.ResponseWriter, r *http.Request) (payload url.Values) {
	v := r.URL.Query()
	if v.Get("sso") == "" {
		return
	}
	payload_bytes, err := base64.StdEncoding.DecodeString(v.Get("sso"))
	if err != nil {
		//TODO: use different error handler
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	sig, err := hex.DecodeString(v.Get("sig"))
	if err != nil {
		//TODO: use different error handler
		http.Error(w, http.StatusText(400), 400)
		return
	}
	mac := hmac.New(sha256.New, []byte(s.config.Discourse["sso-secret"]))
	mac.Write([]byte(v.Get("sso")))
	if !hmac.Equal(mac.Sum(nil), sig) {
		//TODO: use different error handler
		http.Error(w, http.StatusText(400), 400)
		return
	}
	payload, err = url.ParseQuery(string(payload_bytes))
	if err != nil {
		//TODO: use different error handler
		http.Error(w, http.StatusText(400), 400)
		return
	}
	return
}

func (s *Http_server) encode_sso_rsp(nonce string, m *member.Member) (payload, sig string) {
	q := url.Values{}
	q.Set("nonce", nonce)
	q.Set("email", m.Email)
	q.Set("username", m.Username)
	q.Set("require_activation", "true")
	q.Set("external_id", m.Username)
	payload = base64.StdEncoding.EncodeToString([]byte(q.Encode()))
	mac := hmac.New(sha256.New, []byte(s.config.Discourse["sso-secret"]))
	mac.Write([]byte(payload))
	sig = hex.EncodeToString(mac.Sum(nil))
	payload = url.QueryEscape(payload)
	return
}

// sso_handler handles sign-in requests from the talk server, as well as serving
//	sign-in and sign-out responses for local requests
func (s *Http_server) sso_handler() {
	s.mux.HandleFunc("/sso", func(w http.ResponseWriter, r *http.Request) {
		p := s.new_page("sso", "Sign-in")
		p.Session = s.authenticate(r)
		if p.Session != nil && r.PostFormValue("sign-out") == p.Member().Username {
			p.Session.destroy()
			http.SetCookie(w, p.Session.cookie)
			http.Redirect(w, r, "/", 303)
			return
		}
		q := s.parse_sso_req(w, r)
		if q("nonce") != "" && q("return_sso_url") != "" {
			// template adds sso query to form action
			p.Discourse["sso"] = r.URL.RawQuery
		}
		if p.Session == nil {
			if _, ok := r.PostForm["sign-in"]; ok {
				if m := member.Get(r.PostFormValue("username")); m != nil {
					if !m.Authenticate(r.PostFormValue("password")) {
						s.tmpl.Execute(w, p)
						return
					}
					p.Session = s.new_session(m, r.PostFormValue("save-session") == "on")
					http.SetCookie(w, p.Session.cookie)
				} else {
					s.tmpl.Execute(w, p)
					return
				}
			} else if _, ok := r.PostForm["sso"]; !ok {
				s.tmpl.Execute(w, p)
				return
			}
		}
		// Won't reach this point without a successful login
		if q("nonce") != "" && q("return_sso_url") != "" {
			payload, sig := s.encode_sso_rsp(nonce, p.Member())
			http.Redirect(w, r, q.Get("return_sso_url") + "?sso=" + payload + "&sig=" + sig, 303)
		}
		http.Redirect(w, r, "/member", 303)
	})
}
