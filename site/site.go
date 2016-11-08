package main

import (
    "html/template"
    "net/http"
)

var templates = template.Must(template.ParseFiles("template.html"))

type page struct {
}

func handler (w http.ResponseWriter, r *http.Request) {
    p := page{}
    templates.Execute(w, p)
}

func main () {
    http.HandleFunc("/", handler)
    http.ListenAndServe(":8080", nil)
}
