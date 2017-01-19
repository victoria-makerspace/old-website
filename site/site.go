package site

import (
    "html/template"
    _ "encoding/json"
    "log"
    "net/http"
    "net/url"
    _ "crypto/rand"
    _ "golang.org/x/crypto/scrypt"
)

type Http_server struct {
    http.Server
    Mux *http.ServeMux
    Dir string
}

//var tmpl = template.Must(template.ParseGlob(os.Getenv("MAKERSPACE_DIR") + "/site/templates/*"))

type page struct {
    Name string
    Title string
}

func authenticate_form (post url.Values) bool {
    return false
}

func (s *Http_server) root () {
    s.Mux.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/" {
            p := page{"index", ""}
            if r.PostFormValue("signin") == "true" && !authenticate_form(r.PostForm) {
            }
            tmpl := template.Must(template.ParseFiles(s.Dir + "/templates/main.tmpl"))
            tmpl.Execute(w, p)
        } else {
            http.FileServer(http.Dir(s.Dir + "/static/")).ServeHTTP(w, r)
        }
    })
}
/*
func rootHandler (w http.ResponseWriter, r *http.Request) {
    if r.URL.Path == "/" {
        p := page{"index", ""}
        if r.PostFormValue("signin") == "true" && !authenticate_form(r.PostForm) {
        }
        tmpl := template.Must(template.ParseFiles(s.Dir + "/templates/main.tmpl"))
        tmpl.Execute(w, p)
    } else {
        http.FileServer(http.Dir(os.Getenv("MAKERSPACE_DIR") + "/site/static/")).ServeHTTP(w, r)
    }
}

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

func joinHandler (w http.ResponseWriter, r *http.Request) {
    p := page{"join", "Join"}
    tmpl := template.Must(template.ParseFiles(os.Getenv("MAKERSPACE_DIR") + "/site/templates/main.tmpl"))

    tmpl.Execute(w, p)
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
    if (r.URL.Path == "/check") {
        q := r.URL.Query();
        if u, ok := q["username"]; ok {
            if u[0] == "victor" {
                w.Write([]byte("true"))
                return
            }
        } else if e, ok := q["email"]; ok {
            if e[0] == "vvanpoppelen@gmail.com" {
                w.Write([]byte("true"))
                return
            }
        }
        w.Write([]byte("false"))
        return;
    }
}
*/
func (s *Http_server) Serve () {
    s.Mux = http.DefaultServeMux
    s.root()
    log.Panic(s.ListenAndServe())
}
