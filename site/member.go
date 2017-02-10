package site

import (
	"net/http"
	"regexp"
)

func (s *Http_server) member_handler() {
	s.sso_handler()
	s.mux.HandleFunc("/member", func(w http.ResponseWriter, r *http.Request) {
		p := s.new_page("dashboard", "Dashboard")
		p.Session = s.authenticate(w, r)
		if p.Member == nil {
			s.page_error(p, 403, w)
			//http.Error(w, http.StatusText(403), 403)
		}
		s.tmpl.Execute(w, p)
	})
}

func (s *Http_server) tools_handler() {
	s.mux.HandleFunc("/member/tools", func(w http.ResponseWriter, r *http.Request) {
		p := s.new_page("tools", "Tools")
		p.Session = s.authenticate(w, r)
		if p.Member == nil {
			s.page_error(p, 403, w)
			//http.Error(w, http.StatusText(403), 403)
		}
		s.tmpl.Execute(w, p)
	})
}

func (s *Http_server) storage_handler() {
	s.mux.HandleFunc("/member/storage", func(w http.ResponseWriter, r *http.Request) {
		p := s.new_page("storage", "Storage")
		p.Session = s.authenticate(w, r)
		if p.Member == nil {
			s.page_error(p, 403, w)
			//http.Error(w, http.StatusText(403), 403)
		}
		s.tmpl.Execute(w, p)
	})
}

// Avatar returns the url for the member's avatar as determined by the discourse
//	server.
func (p page) Avatar() string {
	rexp := regexp.MustCompile("{size}")
	if t, ok := p.Session.Talk["avatar_template"].([]byte); ok {
		return p.Discourse["url"] + string(rexp.ReplaceAll(t, []byte("120")))
	}
	return ""
}
