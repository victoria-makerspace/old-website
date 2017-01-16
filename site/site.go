package site

import (
    "html/template"
    "log"
    "net/http"
    "os"
)

//var tmpl = template.Must(template.ParseGlob(os.Getenv("MAKERSPACE_DIR") + "/site/templates/*"))

type page struct {
}

func rootHandler (w http.ResponseWriter, r *http.Request) {
    if r.URL.Path == "/" {
//        tmpl = template.Must(template.ParseGlob(os.Getenv("MAKERSPACE_DIR") + "/site/templates/*"))
        tmpl := template.Must(template.ParseFiles(os.Getenv("MAKERSPACE_DIR") + "/site/templates/main.tmpl"))
        tmpl.Execute(w, page{
        })
    } else {
        http.FileServer(http.Dir(os.Getenv("MAKERSPACE_DIR") + "/site/static/")).ServeHTTP(w, r)
    }
}

func memberHandler (w http.ResponseWriter, r *http.Request) {
    //tmpl.Execute(w, page{
    //})
}

func Serve (addr string) {
    http.HandleFunc("/", rootHandler)
    http.HandleFunc("/member", memberHandler)
    log.Panic(http.ListenAndServe(addr, nil))
}
