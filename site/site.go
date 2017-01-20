package site

import (
    "bytes"
    "database/sql"
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
    tmpl template.Template
}

type page struct {
    Name string
    Title string
}

func salt() []byte {
    salt := make([]byte, 32)
    _, err := rand.Read(salt)
    if err != nil { log.Panic(err) }
    return salt
}
func key(password string, salt []byte) []byte {
    key, err := scrypt.Key([]byte(password), salt, 16384, 8, 1, 32);
    if err != nil { log.Panic(err) }
    return key
}

func (s *Http_server) root () {
    s.mux.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" {
            http.FileServer(http.Dir(s.dir + "/static/")).ServeHTTP(w, r)
            return
        }
        p := page{"index", ""}
        tmpl := template.Must(template.ParseFiles(s.dir + "/templates/main.tmpl"))
        tmpl.Execute(w, p)
    })
    s.mux.HandleFunc("/authenticate", func (w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/authenticate" { return };
        r.ParseForm()
        var password struct{
            password_key []byte
            password_salt []byte
        }
        err := s.db.QueryRow("SELECT password_key, password_salt FROM member WHERE username = $1", r.PostForm.Get("username")).Scan(&password)
        rsp := "success"
        if err == sql.ErrNoRows {
            rsp = "invalid username"
        } else {
            if !bytes.Equal(password.password_key, key(r.PostForm.Get("password"), password.password_salt)) {
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
        p := page{"join", "Join"}
        tmpl := template.Must(template.ParseFiles(s.dir + "/templates/main.tmpl"))
        tmpl.Execute(w, p)
    })
    s.mux.HandleFunc("/check", func (w http.ResponseWriter, r *http.Request) {
        if (r.URL.Path == "/check") {
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
    //s.tmpl = template.Must(template.ParseFiles(s.dir + "/templates/main.tmpl"))
    s.root()
    s.join()
    go log.Panic(s.srv.ListenAndServe())
    return s
}
