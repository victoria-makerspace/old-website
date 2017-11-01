package site

import (
	"fmt"
	"github.com/vvanpo/makerspace/member"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"
)

type handler struct {
	paths       []string
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
	"mul": func(i, j uint64) uint64 {
		return i * j
	},
	"div": func(i, j float64) float64 {
		return i / j
	},
	"escape": func(html string) template.HTML {
		return template.HTML(html)
	},
	"now": func() time.Time {
		return time.Now()
	},
	"fmt_time": func(t time.Time) string {
		if t.IsZero() {
			return "never"
		}
		now := time.Now()
		if t.Year() == now.Year() {
			if t.YearDay() == now.YearDay() {
				if diff := now.Sub(t); diff < time.Minute {
					return "just now"
				} else if diff < time.Hour {
					return fmt.Sprintf("%.f minutes ago", diff.Minutes())
				}
				return "today at " + t.Format("3:04 PM")
			}
			return t.Format("Mon, Jan 2")
		}
		return t.Format("Jan 2, 2006")
	},
	"timestamp": func(t int64) time.Time {
		return time.Unix(t, 0)
	},
	"fmt_money": func(amount uint64) string {
		return fmt.Sprintf("$%.2f", float64(amount)/100)
	},
	"Plan_category":   member.Plan_category,
	"Plan_identifier": member.Plan_identifier,
	"Plan_interval":   member.Plan_interval,
}

// tmpl_name is the basename (i.e. minus the ".tmpl") of the template file
func init_handler(name string, handle_func func(*page), paths ...string) {
	var t *template.Template
	// Pages serving only JSON or redirects don't require a template
	tmpl_path := file_path("templates", name+".tmpl")
	if fi, _ := os.Stat(tmpl_path); fi != nil && fi.Mode().IsRegular() {
		t = template.New(name + ".tmpl").Funcs(tmpl_funcmap)
		template.Must(t.ParseFiles(tmpl_path))
	}
	handlers[name] = &handler{paths, handle_func, t}
}

func (hs *http_server) register_handlers() {
	for name, h := range handlers {
		f := func(name string, h *handler) func(w http.ResponseWriter, r *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {
				p := hs.new_page(w, r)
				p.Name = name
				if strings.HasSuffix(r.URL.Path, ".json") {
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
		for _, path := range h.paths {
			hs.Handler.(*http.ServeMux).HandleFunc(path, f)
		}
	}
}
