package site

import (
	"fmt"
	"net/http"
)

type page struct {
	Name  string
	Title string
	Data  map[string]interface{} // Data to be passed to templates or JSON
	Status       int
	*Session
	*Http_server
	http.ResponseWriter
	*http.Request
	cookies		map[string]*http.Cookie
	srv_template bool // srv_json takes precedence over srv_template
	srv_json bool
	redirect     string
}

func (h *Http_server) new_page(w http.ResponseWriter, r *http.Request) *page {
	//// TODO: remove after testing
	h.parse_templates()
	/////
	p := &page{
		Data:           make(map[string]interface{}),
		Status:         http.StatusOK,
		Http_server:    h,
		ResponseWriter: w,
		Request:        r,
		cookies:           make(map[string]*http.Cookie),
		srv_template:   true}
	return p
}

// http_error changes template to error.tmpl, or sets JSON output
func (p *page) http_error(code int) {
	p.Name = "error"
	p.Status = code
	p.Title = fmt.Sprint(code)
	p.Data["error"] = http.StatusText(code)
}
