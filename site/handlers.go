package site

import (
	"encoding/json"
	"log"
	"net/http"
)

var handlers = make(map[string]func(*page))

func (h *Http_server) set_handlers() {
	for path, handler := range handlers {
		h.Handler.(*http.ServeMux).HandleFunc(path, func(w http.ResponseWriter,
			r *http.Request) {
			p := h.new_page(w, r)
			//TODO: recover and do http_error(500)
			handler(p)
			write_rsp(p)
		})
	}
}

func write_rsp(p *page) {
	for _, c := range p.cookies {
		http.SetCookie(p.ResponseWriter, c)
	}
	if p.redirect != "" {
		if p.Status == http.StatusOK {
			p.Status = 303
		}
		http.Redirect(p.ResponseWriter, p.Request, p.redirect, p.Status)
		return
	}
	if p.srv_json {
		j := json.NewEncoder(p.ResponseWriter)
		///TODO: remove after testing, or don't, who cares
		//j.SetIndent("", "    ")
		p.ResponseWriter.Header().Set("Content-Type", "application/json")
		p.WriteHeader(p.Status)
		if err := j.Encode(p.Data); err != nil {
			log.Panicf("JSON encoding error: %q\n", err)
		}
	} else if p.srv_template {
		p.WriteHeader(p.Status)
		if err := p.tmpl.Execute(p.ResponseWriter, p); err != nil {
			log.Panicf("Template parsing error: %q\n", err)
		}
	}
}
