package member

import (
	"fmt"
	"log"
	"net/smtp"
)

func (ms *Members) send_email(from string, to []string, body []byte) {
	config := ms.Config["smtp"].(map[string]interface{})
	auth := smtp.PlainAuth("", config["username"].(string),
		config["password"].(string), config["address"].(string))
	addr := config["address"].(string) +
		fmt.Sprint(int(config["port"].(float64)))
	if err := smtp.SendMail(addr, auth, from, to, body);
		err != nil {
		log.Println("Failed to send email: ", err)
	}
}
