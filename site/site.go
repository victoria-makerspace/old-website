package site

import (
    "database/sql"
    "fmt"
    "html/template"
    "log"
    "net/http"
)

var Templates = [...]string{"main", "index", "sign-in", "join", "dashboard"}

type Config struct {
    Domain string
    Port int
    Templates_dir string
    Static_dir string
    Data_dir string
}

type Http_server struct {
    srv http.Server
    mux *http.ServeMux
    config Config
    db *sql.DB
    tmpl *template.Template
}

type page struct {
    Name string
    Title string
    Member Member
}

func (s *Http_server) root_handler () {
    s.mux.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" {
            http.FileServer(http.Dir(s.config.Static_dir)).ServeHTTP(w, r)
            return
        }
        p := page{Name: "index"}
        s.authenticate(w, r, &p.Member)
        if signout := r.PostFormValue("signout"); signout != "" && signout == p.Member.Username {
            s.sign_out(w, &p.Member)
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

func (s *Http_server) data_handler () {
    s.mux.HandleFunc("/member/data/", func (w http.ResponseWriter, r *http.Request) {
        http.StripPrefix("/member/data/", http.FileServer(http.Dir(s.config.Data_dir))).ServeHTTP(w, r)
    })
}

func (s *Http_server) join_handler () {
    s.mux.HandleFunc("/join", func (w http.ResponseWriter, r *http.Request) {
        p := page{Name: "join", Title: "Join"}
        s.authenticate(w, r, &p.Member)
        if p.Member.Username != "" {
            http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
            return
        }
        s.tmpl.Execute(w, p)
    })
}

func (s *Http_server) parse_templates () {
    s.tmpl = template.Must(template.ParseFiles(func () []string {
        files := make([]string, len(Templates))
        for i := range Templates {
            files[i] = s.config.Templates_dir + Templates[i] + ".tmpl"
        }
        return files
    }()...))
}

func Serve (config Config, db *sql.DB) *Http_server {
    s := new(Http_server)
    s.config = config
    s.srv.Addr = config.Domain + ":" + fmt.Sprint(config.Port)
    s.mux = http.NewServeMux()
    s.srv.Handler = s.mux
    s.db = db
    s.parse_templates()
    s.root_handler()
    s.data_handler()
    s.join_handler()
    s.dashboard_handler()
    go log.Panic(s.srv.ListenAndServe())
    return s
}
