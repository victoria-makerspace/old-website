package billing

import (
	"database/sql"
	"fmt"
	beanstream "github.com/Beanstream/beanstream-go"
	"log"
	"math/rand"
	"time"
)

type Billing struct {
	db       *sql.DB
	config   beanstream.Config
	gateway  beanstream.Gateway
	payments beanstream.PaymentsAPI
	profiles beanstream.ProfilesAPI
}

func Billing_new(merchant_id, payments_api_key, profiles_api_key, reporting_api_key string, db *sql.DB) *Billing {
	rand.Seed(time.Now().UTC().UnixNano())
	b := &Billing{
		db:     db,
		config: beanstream.Config{merchant_id, payments_api_key, profiles_api_key, reporting_api_key, "www", "api", "v1", "-8:00"}}
	b.gateway = beanstream.Gateway{b.config}
	b.payments = b.gateway.Payments()
	b.profiles = b.gateway.Profiles()
	go b.schedule_payments()
	return b
}

func (b *Billing) schedule_payments() {

}

type Profile struct {
	b        *Billing
	username string
	bs       beanstream.Profile
}

func (b *Billing) New_profile(token, name, username string) *Profile {
	p := &Profile{b: b, username: username}
	p.bs.Token = beanstream.Token{
		Token: token,
		Name:  name}
	p.bs.Custom = beanstream.CustomFields{Ref1: username}
	rsp, err := b.profiles.CreateProfile(p.bs)
	if err != nil {
		log.Println(err)
		return nil
	}
	p.bs.Id = rsp.Id
	_, err = b.db.Exec("INSERT INTO billing_profile VALUES ($1, $2)", username, rsp.Id)
	if err != nil {
		log.Panic(err)
		return nil
	}
	return p
}

func (b *Billing) Get_profile(username string) *Profile {
	var id string
	err := b.db.QueryRow("SELECT id FROM billing_profile WHERE username = $1", username).Scan(&id)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return nil
	}
	p := &Profile{b: b, username: username}
	if bs, err := b.profiles.GetProfile(id); err != nil {
		log.Println(err)
		return nil
	} else {
		p.bs = *bs
	}
	return p
}

func (p *Profile) Card() *beanstream.CreditCard {
	if p.bs.Card.Number == "" {
		return nil
	}
	return &p.bs.Card
}

func (p *Profile) Delete_card() {
	if _, err := p.bs.DeleteCard(p.b.profiles, 1); err != nil {
		log.Println(err)
	}
	p.bs.Card = beanstream.CreditCard{}
}

func (p *Profile) Update_card(name, token string) {
	if p.Card() != nil {
		p.Delete_card()
	}
	if _, err := p.b.profiles.AddTokenizedCard(p.bs.Id, name, token); err != nil {
		log.Println(err)
		return
	}
	card, err := p.bs.GetCard(p.b.profiles, 1)
	if err != nil {
		log.Println(err)
		return
	}
	p.bs.Card = *card
}

func (p *Profile) Transaction(amount float32, comment, ip_address string) {
	order_id := fmt.Sprint(rand.Intn(1000000)) + "-" + p.username
	req := beanstream.PaymentRequest{
		PaymentMethod: "payment_profile",
		OrderNumber:   order_id,
		Amount:        amount,
		Profile:       beanstream.ProfilePayment{p.bs.Id, 1, true},
		Comment:       comment,
		CustomerIp:    ip_address,
	}
	rsp, err := p.b.payments.MakePayment(req)
	if err != nil {
		log.Println(err)
	}
	if rsp.Approved != 1 {
		log.Println("Payment of %.2f by %s failed", amount, p.username)
	}
}
