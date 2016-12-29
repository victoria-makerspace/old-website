package site

import (
    "html/template"
    "log"
    "net/http"
)

type page struct {
}

func indexHandler (w http.ResponseWriter, r *http.Request) {
    p := page{}
    template.Execute(w, p)
}

func Serve (addr string) {
    http.HandleFunc("/", indexHandler)
    log.Fatal(http.ListenAndServe(addr, nil))
}
