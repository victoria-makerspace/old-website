package site

import (
    "html/template"
    "log"
    "net/http"
)

var templates = template.Must(template.ParseGlob("site/templates/*"))

type page struct {
}

func indexHandler (w http.ResponseWriter, r *http.Request) {
    p := page{}
    templates.Execute(w, p)
}

func Serve (addr string) {
    http.HandleFunc("/", indexHandler)
    log.Fatal(http.ListenAndServe(addr, nil))
}
