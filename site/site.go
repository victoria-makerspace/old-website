package site

import (
	"database/sql"
	"fmt"
	"github.com/vvanpo/makerspace/billing"
	"github.com/vvanpo/makerspace/member"
	"html/template"
	"log"
	"net/http"
	"path"
	"regexp"
)

var templates = [...]string{"main",
	"error",
	"index",
	"sign-in",
	"join",
	"dashboard",
	"billing",
	"tools",
	"storage"}

func (s *Http_server) parse_templates() {
	s.tmpl = template.Must(template.ParseFiles(func() []string {
		files := make([]string, len(templates))
		for i := range templates {
			files[i] = s.config.Templates_dir + templates[i] + ".tmpl"
		}
		return files
	}()...))
}

type Config struct {
	Domain                              string
	Port                                int
	Templates_dir, Static_dir, Data_dir string
	Discourse                           map[string]string
}

type Http_server struct {
	srv     http.Server
	mux     *http.ServeMux
	config  Config
	db      *sql.DB
	billing *billing.Billing
	tmpl    *template.Template
}

func Serve(config Config, db *sql.DB, b *billing.Billing) *Http_server {
	s := &Http_server{config: config, mux: http.NewServeMux(), db: db, billing: b}
	s.srv.Addr = config.Domain + ":" + fmt.Sprint(config.Port)
	s.srv.Handler = s.mux
	s.parse_templates()
	s.root_handler()
	//s.join_handler()
	//s.classes_handler()
	//s.member_handler()
	go log.Panic(s.srv.ListenAndServe())
	return s
}

type page struct {
	Name    string
	Title   string
	Session *session
	Field   map[string]interface{} // Data to be passed to templates
	http.ResponseWriter
	*http.Request
	*Http_server
}

func (h *Http_server) new_page(name, title string, w http.ResponseWriter, r *http.Request) *page {
	//// TODO: remove after testing
	h.parse_templates()
	/////
	p := &page{Name: name,
		Title:          title,
		Field:          make(map[string]interface{}),
		ResponseWriter: w,
		Request:        r,
		Http_server:    h}
	p.Field["talk_url"] = p.config.Discourse["url"]
	return p
}

func (p *page) Member() *member.Member {
	if p.Session != nil {
		return p.Session.Member
	}
	return nil
}

func (p *page) write_template() {
	if err := p.tmpl.Execute(p.ResponseWriter, p); err != nil {
		log.Println(err)
	}
}

// http_error executes an error template.
func (p *page) http_error(code int) {
	p.Name = "error"
	p.Title = fmt.Sprint(code)
	p.Field["error_message"] = http.StatusText(code)
	p.WriteHeader(code)
	p.write_template()
}

func (h *Http_server) root_handler() {
	h.member_handler()
	h.join_handler()
	h.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("index", "", w, r)
		if r.URL.Path != "/" {
			// Handler for /static/ file directory
			dir := http.Dir(h.config.Static_dir)
			file, err := dir.Open(path.Clean(r.URL.Path))
			if err == nil {
				if fi, err := file.Stat(); err == nil && !fi.IsDir() {
					http.ServeContent(w, r, fi.Name(), fi.ModTime(), file)
					return
				}
			}
			p.authenticate()
			p.http_error(404)
			return
		}
		p.authenticate()
		p.write_template()
	})
}

func (h *Http_server) join_handler() {
	username_rexp := regexp.MustCompile("^[\\pL\\pN\\pM\\pP]+$")
	name_rexp := regexp.MustCompile("^(?:[\\pL\\pN\\pM\\pP]+ ?)+$")
	h.mux.HandleFunc("/join", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("join", "Join", w, r)
		p.authenticate()
		if p.Session != nil {
			p.http_error(403)
			return
		}
		q := r.URL.Query()
		if _, ok := q["exists"]; ok {
			rsp := "true"
			var exists bool
			if _, ok := q["username"]; ok {
				if err := p.db.QueryRow("SELECT true FROM member WHERE username = $1", q.Get("username")).Scan(&exists); err != nil {
					if err != sql.ErrNoRows {
						log.Panic(err)
					}
					rsp = "false"
				}
			} else if _, ok := q["email"]; ok {
				if err := p.db.QueryRow("SELECT true FROM member WHERE email = $1", q.Get("email")).Scan(&exists); err != nil {
					if err != sql.ErrNoRows {
						log.Panic(err)
					}
					rsp = "false"
				}
			} else {
				rsp = "nil"
			}
			w.Write([]byte(rsp))
		} else if _, ok := q["join"]; ok {
			username_length := len([]rune(r.PostFormValue("username")))
			if !username_rexp.MatchString(r.PostFormValue("username")) || username_length > 20 || username_length < 3 {
				//TODO: embed error
			} else if !name_rexp.MatchString(r.PostFormValue("name")) {
				//TODO: embed error
			} else if m := member.New(r.PostFormValue("username"), r.PostFormValue("name"), r.PostFormValue("email"), r.PostFormValue("password"), p.db); m != nil {
				p.new_session(m, true)
				w.Write([]byte("success"))
			}
			return
		} else {
			p.write_template()
		}
	})
}
