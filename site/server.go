package site

import (
	"database/sql"
	"fmt"
	"github.com/vvanpo/makerspace/member"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
)

type Config struct {
	Domain string
	Tls bool
	Port int
	Talk_proxy string
}

func (c Config) Url() string {
	if c.Tls {
		return "https://" + c.Domain
	}
	return "http://" + c.Domain
}

func file_path(path_elem ...string) string {
	gopath := os.Getenv("GOPATH")
	return filepath.Join(gopath, "src", "github.com", "vvanpo", "makerspace",
		"site", filepath.Join(path_elem...))
}

type http_server struct {
	http.Server
	Config
	db          *sql.DB
	header_tmpl *template.Template
	footer_tmpl *template.Template
	error_tmpl  *template.Template
	*member.Members
}

//TODO: set h.ErrorLog to a different logger
func Serve(config Config, members *member.Members, db *sql.DB) {
	hs := &http_server{
		Config:   config,
		db:       db,
		Members:  members}
	hs.Addr = ":" + fmt.Sprint(config.Port)
	hs.Handler = http.NewServeMux()
	hs.header_tmpl = template.Must(template.ParseFiles(file_path("templates",
		"header.tmpl")))
	hs.footer_tmpl = template.Must(template.ParseFiles(file_path("templates",
		"footer.tmpl")))
	hs.error_tmpl = template.Must(template.ParseFiles(file_path("templates",
		"error.tmpl")))
	hs.register_handlers()
	if config.Talk_proxy != "" {
		hs.talk_proxy()
	}
	go log.Panic(hs.ListenAndServe())
}

func (hs *http_server) talk_proxy() {
	u, err := url.Parse(hs.Config.Talk_proxy)
	if err != nil {
		log.Fatal(err)
	}
	rp := httputil.NewSingleHostReverseProxy(u)
	hs.Handler.(*http.ServeMux).Handle("/talk/", rp)
}
