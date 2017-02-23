package site

import (
	"database/sql"
	"fmt"
	"github.com/vvanpo/makerspace/member"
	"github.com/vvanpo/makerspace/talk"
	"html/template"
	"log"
	"net/http"
	"path"
)

var templates = [...]string{"main",
	"header",
	"error",
	"index",
	"sso",
	"join",
	"terms",
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
}

type Http_server struct {
	srv     http.Server
	mux     *http.ServeMux
	config  Config
	db      *sql.DB
	*talk.Talk_api
	*member.Members
	tmpl    *template.Template
}

func Serve(config Config, talk *talk.Talk_api, members *member.Members, db *sql.DB) *Http_server {
	s := &Http_server{
		config: config,
		mux: http.NewServeMux(),
		db: db,
		Talk_api: talk,
		Members: members}
	s.srv.Addr = config.Domain + ":" + fmt.Sprint(config.Port)
	s.srv.Handler = s.mux
	s.parse_templates()
	s.root_handler()
	go log.Panic(s.srv.ListenAndServe())
	return s
}

type page struct {
	Name     string
	Title    string
	*Session
	Field    map[string]interface{} // Data to be passed to templates
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
		//TODO: change to panic?
		log.Println(err)
	}
}

// http_error executes an error template.
func (p *page) http_error(code int) {
	//TODO: pass error code and message via *page object
	p.Name = "error"
	p.Title = fmt.Sprint(code)
	p.Field["error_message"] = http.StatusText(code)
	p.WriteHeader(code)
	//TODO: if content JSON ...write_json() etc
	p.write_template()
}

func (h *Http_server) root_handler() {
	h.member_handler()
	h.join_handler()
	h.terms_handler()
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

//TODO: create talk user
func (h *Http_server) join_handler() {
	h.mux.HandleFunc("/join", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("join", "Join", w, r)
		p.authenticate()
		if p.Session != nil {
			p.http_error(403)
			return
		}
		p.ParseForm()
		if _, ok := p.Form["exists"]; ok {
			rsp := "true"
			var exists bool
			if _, ok := p.Form["username"]; ok {
				if err := p.db.QueryRow("SELECT true FROM member WHERE username = $1", p.FormValue("username")).Scan(&exists); err != nil {
					if err != sql.ErrNoRows {
						log.Panic(err)
					}
					rsp = "false"
				}
			} else if _, ok := p.Form["email"]; ok {
				if err := p.db.QueryRow("SELECT true FROM member WHERE email = $1", p.FormValue("email")).Scan(&exists); err != nil {
					if err != sql.ErrNoRows {
						log.Panic(err)
					}
					rsp = "false"
				}
			} else {
				rsp = "nil"
			}
			w.Write([]byte(rsp))
			return
		} else if _, ok := p.PostForm["join"]; ok {
			//TODO: vary output based on Content-type: application/json or whatever
			p.Check_username(p.PostFormValue("username"))
			/*var check_username map[string]interface{}
			if rsp, err := http.Get(p.Talk_url + "/users/check_username.json?username=" + url.QueryEscape(p.PostFormValue("username"))); err != nil {
				log.Println(err)
			} else if err = json.NewDecoder(rsp.Body).Decode(&check_username); err != nil {
				log.Println(err)
			}
			///TODO: put username/email/name in its own methods and all under one url for the javascript
			if e, ok := check_username["errors"]; ok {
				//TODO: embed errors, given as a JSON string array by discourse
				log.Println(e)
			} else if a, ok := check_username["available"]; ok {
				if a, ok = check_username["available"].(bool); ok && a == true {
					if m := member.New(r.PostFormValue("username"),
						r.PostFormValue("name"), r.PostFormValue("email"),
						r.PostFormValue("password"), p.db); m != nil {
						//TODO: sign in with the talk server immediately, to prevent talk_url errors within p.new_session
						p.new_session(m, true)
						w.Write([]byte("success"))
						return
					}
				}
				//TODO: embed errors
			}
			//TODO: embed "talk is down" error
			p.http_error(500)*/
		}
		p.write_template()
	})
}

func (h *Http_server) terms_handler() {
	h.mux.HandleFunc("/terms", func(w http.ResponseWriter, r *http.Request) {
		p := h.new_page("terms", "Terms & Conditions", w, r)
		p.authenticate()
		if p.Session != nil && p.PostFormValue("agree_to_terms") != "" {
			if _, err := p.db.Exec("UPDATE member SET agreed_to_terms = true "+
				"WHERE username = $1", p.Member().Username); err != nil {
				log.Panic(err)
			}
			p.Member().Agreed_to_terms = true
			http.Redirect(w, r, "/member", 303)
			return
		}
		p.write_template()
	})
}
