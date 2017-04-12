package site

import (
	"database/sql"
	"fmt"
	"github.com/vvanpo/makerspace/member"
	"github.com/vvanpo/makerspace/talk"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
)

func file_path(path_elem ...string) string {
	gopath := os.Getenv("GOPATH")
	return filepath.Join(gopath, "src", "github.com", "vvanpo", "makerspace",
		"site", filepath.Join(path_elem...))
}

type http_server struct {
	http.Server
	config      map[string]interface{}
	db          *sql.DB
	header_tmpl *template.Template
	footer_tmpl *template.Template
	error_tmpl  *template.Template
	*talk.Talk_api
	*member.Members
}

//TODO: set h.ErrorLog to a different logger
func Serve(config map[string]interface{}, talk *talk.Talk_api,
	members *member.Members, db *sql.DB) {
	hs := &http_server{
		config:   config,
		db:       db,
		Talk_api: talk,
		Members:  members}
	hs.Addr = ":" + fmt.Sprint(int(config["port"].(float64)))
	hs.Handler = http.NewServeMux()
	hs.header_tmpl = template.Must(template.ParseFiles(file_path("templates",
		"header.tmpl")))
	hs.footer_tmpl = template.Must(template.ParseFiles(file_path("templates",
		"footer.tmpl")))
	hs.error_tmpl = template.Must(template.ParseFiles(file_path("templates",
		"error.tmpl")))
	hs.register_handlers()
	if u, ok := config["talk-proxy"].(string); ok {
		hs.talk_proxy(u)
	}
	go log.Panic(hs.ListenAndServe())
}

func (hs *http_server) talk_proxy(proxy string) {
	u, err := url.Parse(proxy)
	if err != nil {
		log.Fatal(err)
	}
	rp := httputil.NewSingleHostReverseProxy(u)
	hs.Handler.(*http.ServeMux).Handle("/talk/", rp)
}
