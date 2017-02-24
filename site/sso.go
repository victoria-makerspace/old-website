package site

import (
	"fmt"
	"net/http"
	"net/url"
)

func (p *page) must_authenticate() bool {
	p.authenticate()
	if p.Session == nil {
		p.Name = "sso"
		p.Title = "Sign-in"
		p.WriteHeader(403)
		p.write_template()
		return false
	}
	return true
}

// sso_handler handles sign-in requests from the talk server, as well as serving
//	sign-in and sign-out responses for local requests
func (h *Http_server) sso_handler() {
	h.mux.HandleFunc("/sso.json", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("", "", w, r)
		p.authenticate()
		var response string
		if p.Session == nil {
			r.ParseForm()
			if _, ok := p.PostForm["sign-in"]; ok {
				if m := h.Get_member_by_username(p.PostFormValue("username"));
					m != nil {
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
		p.ParseForm()
		if _, ok := p.PostForm["sign-in"]; ok {
			if m := h.Get_member_by_username(p.PostFormValue("username"));
				m != nil {
				if !m.Authenticate(p.PostFormValue("password")) {
					//TODO: incorrect password, embed error
				} else {
					p.new_session(m, !(p.PostFormValue("save-session") == "on"))
					http.Redirect(w, r, "/member", 303)
				}
			} else {
				//TODO: invalid username, embed error
			}
		}
		q := h.Parse_sso_req(r.URL.Query())
		if q == nil {
			p.write_template()
			return
		}
		sso_payload := q.Get("nonce") != "" && q.Get("return_sso_url") != ""
		if sso_payload {
			p.Field["sso_query"] = r.URL.RawQuery
		}
		if p.Session == nil {
			//TODO: embed return_sso_url
			if _, ok := p.PostForm["sign-in"]; ok {
				if m := h.Get_member_by_username(p.PostFormValue("username"));
					m != nil {
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
			values := url.Values{}
			values.Set("email", p.Member().Email)
			values.Set("username", p.Member().Username)
			values.Set("external_id", fmt.Sprint(p.Member().Id))
			payload, sig := h.Encode_sso_rsp(q.Get("nonce"), values)
			http.Redirect(w, r, q.Get("return_sso_url")+"?sso="+payload+"&sig="+sig, 303)
			return
		}
		http.Redirect(w, r, "/member", 303)
	})
	h.mux.HandleFunc("/sso/sign-out", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("", "", w, r)
		if !p.must_authenticate() {
			return
		}
		return_url := "/"
		if u := p.FormValue("return_path"); u != "" {
			return_path = u
		}
		//TODO: find a secure way to to sign out that works with discourse
		p.destroy_session()
		if t := p.Talk_user(); t != nil {
			t.Logout()
		}
		http.Redirect(w, r, return_path, 303)
	})
}

