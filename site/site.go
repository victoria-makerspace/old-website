package site

import (
    "html/template"
    "log"
    "net/http"
    "net/url"
    "os"
    _ "crypto/rand"
    _ "golang.org/x/crypto/scrypt"
)

//var tmpl = template.Must(template.ParseGlob(os.Getenv("MAKERSPACE_DIR") + "/site/templates/*"))

type page struct {
    Name string
    Title string
}

func authenticate_form (post url.Values) bool {
    return false
}

func rootHandler (w http.ResponseWriter, r *http.Request) {
    if r.URL.Path == "/" {
        p := page{"index", ""}
        if r.PostFormValue("signin") == "true" && !authenticate_form(r.PostForm) {
        }
//        tmpl = template.Must(template.ParseGlob(os.Getenv("MAKERSPACE_DIR") + "/site/templates/*"))
        tmpl := template.Must(template.ParseFiles(os.Getenv("MAKERSPACE_DIR") + "/site/templates/main.tmpl"))
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

func Serve (addr string) {
    http.HandleFunc("/", rootHandler)
    http.HandleFunc("/member", memberHandler)
    http.HandleFunc("/join", joinHandler)
    log.Panic(http.ListenAndServe(addr, nil))
}
