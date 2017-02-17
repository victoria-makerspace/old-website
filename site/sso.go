package site

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"github.com/vvanpo/makerspace/member"
	"net/http"
	"net/url"
)

func (p *page) parse_sso_request() (payload url.Values) {
	v := p.URL.Query()
	if v.Get("sso") == "" {
		return
	}
	payload_bytes, _ := base64.StdEncoding.DecodeString(v.Get("sso"))
	sig, _ := hex.DecodeString(v.Get("sig"))
	mac := hmac.New(sha256.New, []byte(p.config.Discourse["sso-secret"]))
	mac.Write([]byte(v.Get("sso")))
	payload, err := url.ParseQuery(string(payload_bytes))
	if err != nil || !hmac.Equal(mac.Sum(nil), sig) {
		p.http_error(400)
		return nil
	}
	return
}

func (p *page) encode_sso_response(nonce string) (payload, sig string) {
	q := url.Values{}
	q.Set("nonce", nonce)
	q.Set("email", p.Member().Email)
	q.Set("username", p.Member().Username)
	q.Set("external_id", p.Member().Username)
	payload = base64.StdEncoding.EncodeToString([]byte(q.Encode()))
	mac := hmac.New(sha256.New, []byte(p.config.Discourse["sso-secret"]))
	mac.Write([]byte(payload))
	sig = hex.EncodeToString(mac.Sum(nil))
	payload = url.QueryEscape(payload)
	return
}

// sso_handler handles sign-in requests from the talk server, as well as serving
//	sign-in and sign-out responses for local requests
func (h *Http_server) sso_handler() {
	h.mux.HandleFunc("/sso.json", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("sso", "", w, r)
		p.authenticate()
		var response string
		if p.Session == nil {
			r.ParseForm()
			if _, ok := p.PostForm["sign-in"]; ok {
				if m := member.Get(p.PostFormValue("username"), p.db); m != nil {
					if !m.Authenticate(p.PostFormValue("password")) {
						response = "incorrect password"
					} else {
						p.new_session(m, !(p.PostFormValue("save-session") == "on"))
						response = "success"
					}
				} else {
					response = "invalid username"
				}
				p.ResponseWriter.Write([]byte("\"" + response + "\""))
				return
			}
			p.http_error(404)
		} else {
			p.http_error(403)
		}
	})
	h.mux.HandleFunc("/sso", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("sso", "Sign-in", w, r)
		p.authenticate()
		if p.Session != nil && p.PostFormValue("sign-out") == p.Member().Username {
			p.destroy_session()
			http.Redirect(w, r, "/", 303)
			return
		}
		q := p.parse_sso_request()
		if q == nil {
			return
		}
		sso_payload := q.Get("nonce") != "" && q.Get("return_sso_url") != ""
		if sso_payload {
			p.Field["sso_query"] = r.URL.RawQuery
		}
		if p.Session == nil {
			if _, ok := p.PostForm["sign-in"]; ok {
				if m := member.Get(p.PostFormValue("username"), p.db); m != nil {
					if !m.Authenticate(p.PostFormValue("password")) {
						p.write_template()
						return
					}
					p.new_session(m, !(p.PostFormValue("save-session") == "on"))
				} else {
					p.write_template()
					return
				}
			} else {
				p.write_template()
				return
			}
		}
		// Won't reach this point without a successful login
		if sso_payload {
			payload, sig := p.encode_sso_response(q.Get("nonce"))
			http.Redirect(w, r, q.Get("return_sso_url")+"?sso="+payload+"&sig="+sig, 303)
			return
		}
		http.Redirect(w, r, "/member", 303)
	})
}
