package site

import (
	"log"
	"net/http"
	"path"
)

func init() {
	handlers["/"] = root_handler
	handlers["/terms"] = terms_handler
}

func root_handler(p *page) {
	if p.URL.Path != "/" {
		static_handler(p)
		return
	}
	p.Name = "index"
	p.authenticate()
}

func static_handler(p *page) {
	dir := http.Dir(p.config["dir"].(string) + "/site/static/")
	file, err := dir.Open(path.Clean(p.URL.Path))
	if err == nil {
		if f_info, err := file.Stat(); err == nil && !f_info.IsDir() {
			http.ServeContent(p.ResponseWriter, p.Request, f_info.Name(),
				f_info.ModTime(), file)
			p.srv_template = false
			return
		}
	}
	p.http_error(404)
}

func terms_handler(p *page) {
	p.Name = "terms"
	p.Title = "Terms & Conditions"
	p.authenticate()
	if p.Session != nil && p.PostFormValue("agree_to_terms") != "" {
		if _, err := p.db.Exec(
			"UPDATE member "+
				"SET agreed_to_terms = true "+
				"WHERE id = $1",
			p.Member.Id); err != nil {
			log.Panic(err)
		}
		p.Member.Agreed_to_terms = true
		p.Status = 303
		p.redirect = "/member/dashboard"
		return
	}
}
