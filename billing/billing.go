package billing

import (
	"database/sql"
	"fmt"
	beanstream "github.com/Beanstream/beanstream-go"
	"log"
	"math/rand"
	"strconv"
	"time"
)

type Billing struct {
	db       *sql.DB
	config   beanstream.Config
	gateway  beanstream.Gateway
	payments beanstream.PaymentsAPI
	profiles beanstream.ProfilesAPI
	reports	 beanstream.ReportsAPI
}

func Billing_new(merchant_id, payments_api_key, profiles_api_key, reports_api_key string, db *sql.DB) *Billing {
	rand.Seed(time.Now().UTC().UnixNano())
	b := &Billing{
		db:     db,
		config: beanstream.Config{merchant_id, payments_api_key, profiles_api_key, reports_api_key, "www", "api", "v1", "-8:00"}}
	b.gateway = beanstream.Gateway{b.config}
	b.payments = b.gateway.Payments()
	b.profiles = b.gateway.Profiles()
	b.reports = b.gateway.Reports()
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
	}
	p.bs.Id = rsp.Id
	_, err = b.db.Exec("INSERT INTO billing_profile VALUES ($1, $2)", username, rsp.Id)
	if err != nil {
		log.Panic(err)
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

func prorate_month(amount float64) float64 {
	end_of_month := time.Now().AddDate(0, 1, 0 - time.Now().Day()).Day()
	prorated := float64(end_of_month - time.Now().Day()) * amount
	prorated /= float64(end_of_month)
	return prorated
}

func prorate_month_start(amount float64) float64 {
	prorated := float64(time.Now().Day()) * amount
	return prorated
}

func (p *Profile) Update_billing(name string, amount float64) {
	var id int
	var a string
	err := p.b.db.QueryRow("SELECT id, amount FROM billing WHERE username = $1 AND name == $2", p.username, name).Scan(&id, &a)
	if err == sql.ErrNoRows {
		// Register member
		_, err = p.b.db.Exec("INSERT INTO billing (username, name, amount) VALUES ($1, $2, $3)", p.username, name, amount)
		if err != nil {
			log.Panic(err)
		}
		// Pro-rate the current month's bill, do transaction immediately.
		p.New_transaction(prorate_month(amount), name, "")
		return
	} else if err != nil {
		log.Panic(err)
	}
	prev_amount, err := strconv.ParseFloat(a[1:], 32)
	if err != nil {
		log.Panic(err)
	}
	// If billing already exists and the amount hasn't changed, do nothing.
	if prev_amount == amount {
		return
	}
	// If a billing exists but the amount needs to be updated, expire the existing
	//	billing on the 1st of this month, pro-rate the transaction for the current
	//  month, and start a new billing for next month.
	end_date := time.Now().AddDate(0, 0, 1 - time.Now().Day())
	_, err = p.b.db.Exec("UPDATE billing SET end_date = $1)", end_date)
	if err != nil {
		log.Panic(err)
	}
	_, err = p.b.db.Exec("INSERT INTO billing (username, name, amount) VALUES ($1, $2, $3)", p.username, name, amount)
	if err != nil {
		log.Panic(err)
	}
	prorated := prorate_month(amount) + prorate_month_start(prev_amount)
	// Do transaction for this month with prorated amount
	p.New_transaction(prorated, name, "")
}

type Transaction struct {
	id string
	username string
	Date time.Time
	Approved bool
	Order_id string
	Amount float64
	Name string	// "Membership dues", "Storage fees", etc.
	Card string	// Last 4 digits
	Ip_address string
	billing_id int
}

func (p *Profile) New_transaction(amount float64, name, ip_address string) *Transaction {
	order_id := fmt.Sprint(rand.Intn(1000000)) + "-" + p.username
	req := beanstream.PaymentRequest{
		PaymentMethod: "payment_profile",
		OrderNumber:   order_id,
		Amount:        float32(amount),
		Profile:       beanstream.ProfilePayment{p.bs.Id, 1, true},
		Comment:       name,
		CustomerIp:    ip_address,
	}
	rsp, err := p.b.payments.MakePayment(req)
	if err != nil {
		log.Println(err)
	}
	if !rsp.IsApproved() {
		log.Println("Payment of %.2f by %s failed", amount, p.username)
	}
	txn := &Transaction{id: rsp.ID,
			username: p.username,
			Date: time.Now(),
			Approved: rsp.IsApproved(),
			Order_id: rsp.OrderNumber,
			Amount: amount,
			Name: name,
			Card: rsp.Card.LastFour,
			Ip_address: ip_address}
	_, err = p.b.db.Exec("INSERT INTO transaction (id, username, approved, order_id, amount, name, card, ip_address) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", rsp.ID, p.username, txn.Approved, txn.Order_id, txn.Amount, txn.Name, txn.Card, txn.Ip_address)
	if err != nil {
		log.Panic(err)
	}
	return txn
}

func (p *Profile) Get_transactions(number int) []*Transaction {
	var txns []*Transaction
	rows, err := p.b.db.Query("SELECT id, approved, order_id, amount, name, card, ip_address, time FROM transaction WHERE username = $1 LIMIT $2", p.username, number)
	defer rows.Close()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return txns
	}
	for i := 0; rows.Next(); i++ {
		txn := &Transaction{username: p.username}
		txns = append(txns, txn)
		var (
			amount	string
			name	sql.NullString
			card	sql.NullString
			ip_address	sql.NullString
		)
		if err := rows.Scan(&txn.id, &txn.Approved, &txn.Order_id, &amount, &name, &card, &ip_address, &txn.Date); err != nil {
			log.Panic(err)
		}
		if txn.Amount, err = strconv.ParseFloat(amount[1:], 32); err != nil {
			log.Println(err)
		}
		txn.Name = name.String
		txn.Card = card.String
		txn.Ip_address = ip_address.String
	}
	if err := rows.Err(); err != nil {
		log.Panic(err)
	}
	return txns
}
