package site

import (
	"html/template"
	"net/http"
	"strings"
	"os"
)

type handler struct {
	path        string
	handle_func func(*page)
	*template.Template
}

var handlers = make(map[string]*handler)

var tmpl_funcmap = template.FuncMap{
	"add": func(i, j int) int {
		return i + j
	},
	"sub": func(i, j int) int {
		return i - j
	},
}

// tmpl_name is the basename (i.e. minus the ".tmpl") of the template file
func init_handler(path, name string, handle_func func(*page)) {
	var t *template.Template
	// Pages serving only JSON or redirects don't require a template
	tmpl_path := file_path("templates", name+".tmpl")
	if fi, _ := os.Stat(tmpl_path); fi != nil && fi.Mode().IsRegular() {
		t = template.New(name+".tmpl").Funcs(tmpl_funcmap)
		template.Must(t.ParseFiles(tmpl_path))
	}
	handlers[name] = &handler{path, handle_func, t}
}

func (hs *http_server) register_handlers() {
	for name, h := range handlers {
		f := func(name string, h *handler) func(w http.ResponseWriter, r *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {
				p := hs.new_page(w, r)
				p.Name = name
				if strings.HasSuffix(h.path, ".json") {
					p.srv_json = true
				}
				p.tmpl = h.Template
				p.ParseForm()
				// Don't authenticate on static requests
				if name != "index" {
					p.authenticate()
				}
				//TODO: recover and do http_error(500)
				h.handle_func(p)
				p.write_response()
			}
		}(name, h)
		hs.Handler.(*http.ServeMux).HandleFunc(h.path, f)
	}
}
