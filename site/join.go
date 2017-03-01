package site

import (
	"log"
)

func init() {
	handlers["/join"] = join_handler
}

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
	name := p.PostFormValue("name")
	p.Data["username"] = username
	p.Data["email"] = email
	p.Data["name"] = name
	m, err := p.New_member(username, name, email, p.PostFormValue("password"))
	if m == nil {
		for k, v := range err {
			p.Data[k] = v
		}
		return
	}
	talk_user := m.Sync(m.Id, m.Username, m.Email, m.Name)
	if talk_user == nil {
		m.Delete_member()
		log.Println("Talk sync_sso failed on new member: ", m)
		p.http_error(500)
		return
	}
	log.Printf("New member: (%d) %s\n", m.Id, username)
	if talk_user.Active {
		m.Activate()
		p.new_session(m, true)
		p.redirect = "/member/dashboard"
		return
	}
	//TODO: implement own e-mail validation
	talk_user.Send_activation_email()
	p.redirect = "/member/join/activate"
}
