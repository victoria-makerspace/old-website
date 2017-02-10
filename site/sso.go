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
func (h *Http_server) sso_handler() {
	h.mux.HandleFunc("/sso.json", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("sso", "")
		p.Session = h.authenticate(w, r)
		var response string
		if p.Session == nil {
			r.ParseForm()
			if _, ok := r.PostForm["sign-in"]; ok {
				if m := member.Get(r.PostFormValue("username"), h.db); m != nil {
					if !m.Authenticate(r.PostFormValue("password")) {
						response = "incorrect password"
					} else {
						p.Session = h.new_session(w, m, r.PostFormValue("save-session") == "on")
						response = "success"
					}
				} else {
					response = "invalid username"
				}
				w.Write([]byte("\"" + response + "\""))
				return
			}
			http.Error(w, http.StatusText(404), 404)
		} else {
			http.Error(w, http.StatusText(403), 403)
		}
	})
	h.mux.HandleFunc("/sso", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("sso", "Sign-in")
		p.Session = h.authenticate(w, r)
		if p.Session != nil && r.PostFormValue("sign-out") == p.Member().Username {
			p.Session.destroy(w)
			http.SetCookie(w, p.Session.cookie)
			http.Redirect(w, r, "/", 303)
			return
		}
		q := h.parse_sso_req(w, r)
		sso_payload := q.Get("nonce") != "" && q.Get("return_sso_url") != ""
		if sso_payload {
			// template adds sso query to form action
			p.Discourse["sso"] = r.URL.RawQuery
		}
		if p.Session == nil {
			if _, ok := r.PostForm["sign-in"]; ok {
				if m := member.Get(r.PostFormValue("username"), h.db); m != nil {
					if !m.Authenticate(r.PostFormValue("password")) {
						h.tmpl.Execute(w, p)
						return
					}
					p.Session = h.new_session(w, m, r.PostFormValue("save-session") == "on")
				} else {
					h.tmpl.Execute(w, p)
					return
				}
			} else {
				h.tmpl.Execute(w, p)
				return
			}
		}
		// Won't reach this point without a successful login
		if sso_payload {
			payload, sig := h.encode_sso_rsp(q.Get("nonce"), p.Member())
			http.Redirect(w, r, q.Get("return_sso_url") + "?sso=" + payload + "&sig=" + sig, 303)
		}
		http.Redirect(w, r, "/member", 303)
	})
}
