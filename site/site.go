package site

import (
    "database/sql"
    "html/template"
    "log"
    "net/http"
)

var Templates = [...]string{"main", "index", "join"}

type Http_server struct {
    srv http.Server
    mux *http.ServeMux
    domain string
    dir string
    db *sql.DB
    tmpl *template.Template
}

type page struct {
    Name string
    Title string
    Member Member
}

func (s *Http_server) root () {
    s.mux.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {

s.parse_templates()

        if r.URL.Path != "/" {
            http.FileServer(http.Dir(s.dir + "/static/")).ServeHTTP(w, r)
            return
        }
        p := page{Name: "index"}
        if r.PostFormValue("signin") == "true" {
            if username, password := s.sign_in(w, r); username && password {
            }
        } else {
            s.authenticate(w, r, &p.Member)
            if signout := r.PostFormValue("signout"); signout != "" && signout == p.Member.Username {
                s.sign_out(w, &p.Member)
            }
        }
        s.tmpl.Execute(w, p)
    })
    s.mux.HandleFunc("/exists", func (w http.ResponseWriter, r *http.Request) {
        if (r.URL.Path == "/exists") {
            q := r.URL.Query()
            rsp := "nil"
            if _, ok := q["username"]; ok {
                var n int
                err := s.db.QueryRow("SELECT COUNT(*) FROM member WHERE username = $1", q.Get("username")).Scan(&n)
                if err != nil { log.Panic(err) }
                if n == 0 {
                    rsp = "false"
                } else { rsp = "true" }
            } else if _, ok := q["email"]; ok {
                var n int
                err := s.db.QueryRow("SELECT COUNT(*) FROM member WHERE email = $1", q.Get("email")).Scan(&n)
                if err != nil { log.Panic(err) }
                if n == 0 {
                    rsp = "false"
                } else { rsp = "true" }
            }
            w.Write([]byte(rsp))
        }
    })
}

func (s *Http_server) join () {
    s.mux.HandleFunc("/join", func (w http.ResponseWriter, r *http.Request) {
        p := page{Name: "join", Title: "Join"}
        s.tmpl.Execute(w, p)
    })
}

func (s *Http_server) parse_templates () {
    s.tmpl = template.Must(template.ParseFiles(func () []string {
        files := make([]string, len(Templates))
        for i := range Templates {
            files[i] = s.dir + "/templates/" + Templates[i] + ".tmpl"
        }
        return files
    }()...))
}

func Serve (domain, address, dir string, db *sql.DB) *Http_server {
    s := new(Http_server)
    s.domain = domain
    s.srv.Addr = address
    s.mux = http.NewServeMux()
    s.srv.Handler = s.mux
    s.dir = dir
    s.db = db
    s.parse_templates()
    s.root()
    s.signin()
    s.join()
    go log.Panic(s.srv.ListenAndServe())
    return s
}
