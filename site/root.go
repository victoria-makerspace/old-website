package site

import (
	"log"
	"net/http"
	"path"
)

func init() {
	init_handler("index", root_handler, "/")
	init_handler("terms", terms_handler, "/terms")
}

func root_handler(p *page) {
	if p.URL.Path != "/" {
		p.tmpl = nil
		static_handler(p)
		return
	}
	// See register_handlers(), need to explicitly authenticate here because we
	//	don't want to authenticate for static requests
	p.authenticate()
}

func static_handler(p *page) {
	dir := http.Dir(file_path("static"))
	file, err := dir.Open(path.Clean(p.URL.Path))
	if err == nil {
		if f_info, err := file.Stat(); err == nil && !f_info.IsDir() {
			http.ServeContent(p.ResponseWriter, p.Request, f_info.Name(),
				f_info.ModTime(), file)
			return
		}
	}
	p.authenticate()
	p.http_error(404)
}

func terms_handler(p *page) {
	p.Title = "Terms & Conditions"
	if p.Session != nil && p.PostFormValue("agree_to_terms") != "" {
		//TODO: put in member package
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
