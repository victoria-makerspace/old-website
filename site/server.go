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
	"verify-email",
	"join",
	"terms",
	"dashboard",
	"account",
	"billing",
	"members",
	"tools",
	"storage",
	"admin"}

func (h *http_server) parse_templates() {
	h.tmpl = template.New("main.tmpl").Funcs(template.FuncMap{
		"add": func(i, j int) int {
			return i + j
		},
		"sub": func(i, j int) int {
			return i - j
		},
	})
	template.Must(h.tmpl.ParseFiles(func() []string {
		files := make([]string, len(templates))
		for i := range templates {
			files[i] = h.config["dir"].(string) + "/site/templates/" +
				templates[i] + ".tmpl"
		}
		return files
	}()...))
}

type http_server struct {
	http.Server
	config map[string]interface{}
	db     *sql.DB
	tmpl   *template.Template
	*talk.Talk_api
	*member.Members
}

//TODO: set h.ErrorLog to a different logger
func Serve(config map[string]interface{}, talk *talk.Talk_api, members *member.Members, db *sql.DB) {
	h := &http_server{
		config:   config,
		db:       db,
		Talk_api: talk,
		Members:  members}
	h.Addr = config["domain"].(string) + ":" +
		fmt.Sprint(int(config["port"].(float64)))
	h.Handler = http.NewServeMux()
	h.parse_templates()
	h.set_handlers()
	go log.Panic(h.ListenAndServe())
}
