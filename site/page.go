package site

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

type page struct {
	Name  string
	Title string
	Data  map[string]interface{} // Data to be passed to templates or JSON
	// HTTP status code
	Status int
	// If Session is non-nil, the user is authenticated
	*Session
	*http_server
	http.ResponseWriter
	*http.Request
	cookies map[string]*http.Cookie
	// srv_json encodes the Data structure as JSON and writes it to the response
	srv_json bool
	// Template to serve between header and footer templates, if srv_json flag
	//	isn't set
	tmpl     *template.Template
	redirect string // redirect takes precedence over srv_json and tmpl
}

func (hs *http_server) new_page(w http.ResponseWriter, r *http.Request) *page {
	return &page{
		Data:           make(map[string]interface{}),
		Status:         http.StatusOK,
		http_server:    hs,
		ResponseWriter: w,
		Request:        r,
		cookies:        make(map[string]*http.Cookie)}
}

// http_error changes template to error.tmpl, or sets JSON output
// 	Careful to scrub output of extraneous p.Data values, if srv_json is set
func (p *page) http_error(code int) {
	p.tmpl = p.http_server.error_tmpl
	p.Name = "error"
	p.Title = fmt.Sprint(code)
	p.Status = code
	p.Data["status"] = code
	p.Data["error"] = http.StatusText(code)
	p.redirect = "" // Cancel any pending redirect
}

func (p *page) write_response() {
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
		j.SetIndent("", "    ")
		p.ResponseWriter.Header().Set("Content-Type", "application/json")
		p.WriteHeader(p.Status)
		if err := j.Encode(p.Data); err != nil {
			log.Panicf("JSON encoding error: %q\n", err)
		}
	} else if p.tmpl != nil {
		p.WriteHeader(p.Status)
		// Sandwich p.tmpl between header and footer
		for _, t := range []*template.Template{p.header_tmpl, p.tmpl,
			p.footer_tmpl} {
			if err := t.Execute(p.ResponseWriter, p); err != nil {
				log.Panicf("Template execution error: %q\n", err)
			}
		}
	}
}
