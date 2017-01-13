package site

import (
    "html/template"
    "log"
    "net/http"
)

var templates = template.Must(template.ParseGlob("site/templates/*"))

type page struct {
}

func rootHandler (w http.ResponseWriter, r *http.Request) {
    if r.URL.Path == "/" {
        p := page{}
        templates.Execute(w, p)
    } else {
        http.FileServer(http.Dir("site/static/")).ServeHTTP(w, r)
    }
}

func Serve (addr string) {
    http.HandleFunc("/", rootHandler)
    log.Panic(http.ListenAndServe(addr, nil))
}
