package site

import (
	"log"
)

func init() {
	handlers["/join"] = join_handler
}

//TODO: create talk user
func join_handler(p *page) {
	p.Name = "join"
	p.Title = "Join"
	p.authenticate()
	if p.Session != nil {
		p.http_error(403)
		return
	}
	p.ParseForm()
	if _, ok := p.PostForm["join"]; !ok {
		return
	}
	username := p.PostFormValue("username")
	email := p.PostFormValue("email")
	m, err := p.New_member(username, p.PostFormValue("name"), email,
		p.PostFormValue("password"))
	if m == nil {
		for k, v := range err {
			p.Data[k] = v
		}
		return
	}
	//TODO: only sign-in and create talk user once email has been verified
	log.Println(m.Sync(m.Id, m.Username, m.Email, m.Name))
}
