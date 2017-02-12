package site

import (
	"net/http"
)

func (h *Http_server) member_handler() {
	h.sso_handler()
	h.billing_handler()
	h.mux.HandleFunc("/member", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("dashboard", "Dashboard", w, r)
		p.authenticate()
		if p.Session == nil {
			p.http_error(403)
			return
		}
		p.write_template()
	})
}

func (h *Http_server) tools_handler() {
	h.mux.HandleFunc("/member/tools", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("tools", "Tools", w, r)
		p.authenticate()
		if p.Session == nil {
			p.http_error(403)
			return
		}
		p.write_template()
	})
}

func (h *Http_server) storage_handler() {
	h.mux.HandleFunc("/member/storage", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("storage", "Storage", w, r)
		p.authenticate()
		if p.Session == nil {
			p.http_error(403)
			return
		}
		p.write_template()
	})
}
