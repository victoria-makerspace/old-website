package member

import (
	"encoding/base64"
	"fmt"
	"log"
	"mime"
	"net/mail"
	"net/smtp"
	"time"
)

type message struct {
	from    mail.Address
	to      []mail.Address
	cc      []mail.Address
	bcc     []mail.Address
	subject string
	body    string
}

func (msg *message) set_from(name, email string) {
	msg.from = mail.Address{name, email}
}
func (msg *message) add_to(name, email string) {
	addr := mail.Address{name, email}
	msg.to = append(msg.to, addr)
}
func (msg *message) add_cc(name, email string) {
	addr := mail.Address{name, email}
	msg.cc = append(msg.cc, addr)
}
func (msg *message) add_bcc(name, email string) {
	addr := mail.Address{name, email}
	msg.bcc = append(msg.bcc, addr)
}
func (msg *message) emails() []string {
	emails := make([]string, len(msg.to)+len(msg.cc)+len(msg.bcc))
	for i, a := range append(msg.to, append(msg.cc, msg.bcc...)...) {
		emails[i] = a.Address
	}
	return emails
}
func (ms *Members) format_message(msg message) []byte {
	enc := func(s string) string {
		return mime.BEncoding.Encode("utf-8", s)
	}
	// RFC5322 date format
	now := time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	body := "Date: " + now + "\r\n"
	body += "From: " + msg.from.String() + "\r\n"
	if len(msg.to) > 0 {
		body += "To: " + msg.to[0].String()
		for _, to := range msg.to[1:] {
			body += ", " + to.String()
		}
		body += "\r\n"
	}
	if len(msg.cc) > 0 {
		body += "Cc: " + msg.cc[0].String()
		for _, cc := range msg.cc[1:] {
			body += ", " + cc.String()
		}
		body += "\r\n"
	}
	body += "Subject: " + enc(ms.Config.Smtp.Subject_prefix+msg.subject) +
		"\r\n"
	body += "Content-Type: text/plain; charset=\"utf-8\"\r\n"
	body += "Content-Transfer-Encoding: base64\r\n"
	body += "\r\n" + base64.StdEncoding.EncodeToString([]byte(msg.body))
	return []byte(body)
}

//TODO: log e-mail
func (ms *Members) send_email(from string, to []string, body []byte) {
	auth := smtp.PlainAuth("", ms.Config.Smtp.Username, ms.Config.Smtp.Password,
		ms.Config.Smtp.Address)
	addr := ms.Config.Smtp.Address + ":" + fmt.Sprint(ms.Config.Smtp.Port)
	go func() {
		if err := smtp.SendMail(addr, auth, from, to, body); err != nil {
			log.Println("Failed to send e-mail: ", err)
		}
	}()
}
