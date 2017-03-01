package site

import (
	"database/sql"
	"fmt"
	"github.com/vvanpo/makerspace/member"
	"github.com/vvanpo/makerspace/talk"
	"html/template"
	"log"
	"net/http"
)

var templates = [...]string{
	"main",
	"header",
	"error",
	"index",
	"sso",
	"reset-password",
	"join",
	"terms",
	"dashboard",
	"preferences",
	"billing",
	"tools",
	"storage"}

func (h *Http_server) parse_templates() {
	h.tmpl = template.Must(template.ParseFiles(func() []string {
		files := make([]string, len(templates))
		for i := range templates {
			files[i] = h.config.Templates_dir + templates[i] + ".tmpl"
		}
		return files
	}()...))
}

type Config struct {
	Domain                              string
	Port                                int
	Templates_dir, Static_dir, Data_dir string
}

type Http_server struct {
	http.Server
	config Config
	db     *sql.DB
	*talk.Talk_api
	*member.Members
	tmpl *template.Template
}

//TODO: set h.ErrorLog to a different logger
func Serve(config Config, talk *talk.Talk_api, members *member.Members, db *sql.DB) *Http_server {
	h := &Http_server{
		config:   config,
		db:       db,
		Talk_api: talk,
		Members:  members}
	h.Addr = config.Domain + ":" + fmt.Sprint(config.Port)
	h.Handler = http.NewServeMux()
	h.parse_templates()
	h.set_handlers()
	go log.Panic(h.ListenAndServe())
	return h
}
