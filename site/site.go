package site

import (
    "database/sql"
    "encoding/hex"
    "html/template"
    "log"
    "net/http"
    "crypto/rand"
    "golang.org/x/crypto/scrypt"
)

type Http_server struct {
    srv http.Server
    mux *http.ServeMux
    dir string
    db *sql.DB
    tmpl *template.Template
}

type page struct {
    Name string
    Title string
    Member struct {
        Authenticated bool
        Username string
    }
}

func salt () string {
    salt := make([]byte, 32)
    _, err := rand.Read(salt)
    if err != nil { log.Panic(err) }
    return hex.EncodeToString(salt)
}
func key (password, salt string) string {
    s, err := hex.DecodeString(salt)
    if err != nil { log.Panicf("Invalid salt: %s", err) }
    key, err := scrypt.Key([]byte(password), s, 16384, 8, 1, 32);
    if err != nil { log.Panic(err) }
    return hex.EncodeToString(key)
}

func (s *Http_server) root () {
    s.mux.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" {
            http.FileServer(http.Dir(s.dir + "/static/")).ServeHTTP(w, r)
            return
        }
        p := page{Name: "index"}
        s.tmpl.Execute(w, p)
    })
    s.mux.HandleFunc("/authenticate", func (w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/authenticate" { return };
        r.ParseForm()
        var password struct{
            key string
            salt string
        }
        err := s.db.QueryRow("SELECT password_key, password_salt FROM member WHERE username = $1", r.PostForm.Get("username")).Scan(&password.key, &password.salt)
        rsp := "success"
        if err == sql.ErrNoRows {
            rsp = "invalid username"
        } else {
            if password.key != key(r.PostForm.Get("password"), password.salt) {
                rsp = "incorrect password"
            } else {
                http.SetCookie(w, &http.Cookie{Name: "session", Value: r.PostForm.Get("username")})
            }
        }
        w.Write([]byte("\"" + rsp + "\""))
    })
}

func (s *Http_server) join () {
    s.mux.HandleFunc("/join", func (w http.ResponseWriter, r *http.Request) {
        p := page{Name: "join", Title: "Join"}
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
                err := s.db.QueryRow("SELECT COUNT(*) FROM email WHERE address = $1", q.Get("email")).Scan(&n)
                if err != nil { log.Panic(err) }
                if n == 0 {
                    rsp = "false"
                } else { rsp = "true" }
            }
            w.Write([]byte(rsp))
        }
    })
}

/*

func memberHandler (w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFiles(os.Getenv("MAKERSPACE_DIR") + "/site/templates/main.tmpl"))
    tmpl.Execute(w, page{
        "member",
        "Dashboard",
    })
}

*/
func Serve (address, dir string, db *sql.DB) *Http_server {
    s := new(Http_server)
    s.srv.Addr = address
    s.mux = http.NewServeMux()
    s.srv.Handler = s.mux
    s.dir = dir
    s.db = db
    s.tmpl = template.Must(template.ParseFiles(s.dir + "/templates/main.tmpl"))
    s.root()
    s.join()
    go log.Panic(s.srv.ListenAndServe())
    return s
}
