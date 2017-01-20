package site

import (
    "html/template"
    "log"
    "net/http"
    "net/url"
    _ "crypto/rand"
    _ "golang.org/x/crypto/scrypt"
)

type Http_server struct {
    srv http.Server
    mux *http.ServeMux
    dir string
    tmpl template.Template
}

type page struct {
    Name string
    Title string
}

func authenticate_form (post url.Values) bool {
    return false
}

func (s *Http_server) root () {
    s.mux.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" {
            http.FileServer(http.Dir(s.dir + "/static/")).ServeHTTP(w, r)
            return
        }
        p := page{"index", ""}
        if r.PostFormValue("signin") == "true" && !authenticate_form(r.PostForm) {
        }
        tmpl := template.Must(template.ParseFiles(s.dir + "/templates/main.tmpl"))
        tmpl.Execute(w, p)
    })
    s.mux.HandleFunc("/authenticate", func (w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/authenticate" { return };
        r.ParseForm()
        username := r.PostForm.Get("username");
        password := r.PostForm.Get("password");
        rsp := "success"
        if username != "victor" {
            rsp = "invalid username"
        } else if password != "abc" {
            rsp = "incorrect password"
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
            q := r.URL.Query();
            rsp := "nil"
            if u, ok := q["username"]; ok {
                if rsp = "false"; u[0] == "victor" {
                    rsp = "true"
                }
            } else if e, ok := q["email"]; ok {
                if rsp = "false"; e[0] == "vvanpoppelen@gmail.com" {
                    rsp = "true"
                }
            }
            w.Write([]byte(rsp))
        }
    })
}

/*
    // salt := make([]byte, 16)
    // _, err := rand.Read(salt)
    // if err != nil {}
    // scrypt.Key([]byte(password), salt, 16384, 8, 1, 32);

func memberHandler (w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFiles(os.Getenv("MAKERSPACE_DIR") + "/site/templates/main.tmpl"))
    tmpl.Execute(w, page{
        "member",
        "Dashboard",
    })
}

*/
func Serve (address, dir string) *Http_server {
    s := new(Http_server)
    s.srv.Addr = address
    s.dir = dir
    s.mux = http.NewServeMux()
    s.srv.Handler = s.mux
    s.root()
    s.join()
    //s.tmpl = template.Must(template.ParseFiles(s.dir + "/templates/main.tmpl"))
    go log.Panic(s.srv.ListenAndServe())
    return s
}
