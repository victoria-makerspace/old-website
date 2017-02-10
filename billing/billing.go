package billing

import (
	"database/sql"
	"fmt"
	beanstream "github.com/Beanstream/beanstream-go"
	"github.com/lib/pq"
	"github.com/vvanpo/makerspace/member"
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
	reports  beanstream.ReportsAPI
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
	go b.payment_scheduler()
	return b
}

type student struct {
	Institution string
	Graduation_date time.Time
}

type Profile struct {
	member	 *member.Member
	b        *Billing
	bs       beanstream.Profile
	Student  *student
}

func (b *Billing) New_profile(token, cardholder string, m *member.Member) *Profile {
	p := &Profile{b: b, member: m}
	p.bs.Token = beanstream.Token{
		Token: token,
		Name:  cardholder}
	p.bs.Custom = beanstream.CustomFields{Ref1: m.Username}
	rsp, err := b.profiles.CreateProfile(p.bs)
	if err != nil {
		log.Println(err)
	}
	p.bs.Id = rsp.Id
	_, err = b.db.Exec("INSERT INTO billing_profile VALUES ($1, $2)", m.Username, rsp.Id)
	if err != nil {
		log.Panic(err)
	}
	return p
}

func (b *Billing) Get_profile(m *member.Member) *Profile {
	var (
		id string
		username, institution sql.NullString
		grad_date pq.NullTime
	)
	err := b.db.QueryRow("SELECT bp.id, s.username, s.institution, s.graduation_date FROM billing_profile bp LEFT JOIN student s USING (username) WHERE bp.username = $1", m.Username).Scan(&id, &username, &institution, &grad_date)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return nil
	}
	p := &Profile{b: b, member: m}
	if bs, err := b.profiles.GetProfile(id); err != nil {
		log.Println(err)
		return nil
	} else {
		p.bs = *bs
		if username.Valid {
			p.Student = &student{institution.String, grad_date.Time}
		}
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
	days_in_month := first_of_next_month().AddDate(0, 0, -1).Day()
	days_left := days_in_month - time.Now().Day()
	return amount * float64(days_left) / float64(days_in_month)
}

func (p *Profile) Update_billing(name string, amount float64) {
	var id int
	var a string
	var start_date time.Time
	err := p.b.db.QueryRow("SELECT id, amount, start_date FROM billing WHERE username = $1 AND name = $2 AND (end_date > now() OR end_date IS NULL)", p.member.Username, name).Scan(&id, &a, &start_date)
	if err == sql.ErrNoRows {
		// Register billing
		_, err = p.b.db.Exec("INSERT INTO billing (username, name, amount) VALUES ($1, $2, $3)", p.member.Username, name, amount)
		if err != nil {
			log.Panic(err)
		}
		// Prorate the current month's bill, do transaction immediately.
		p.New_transaction(prorate_month(amount), name + " (prorated)", "")
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
	// If a billing exists but the amount needs to be updated, expire the
	//	existing billing at now(), and create a new billing with the same start
	//	date as the old one (so that there is no confusion about start date when
	//	looking at the list of billings).
	p.Cancel_billing(name)
	_, err = p.b.db.Exec("INSERT INTO billing (username, name, amount, start_date) VALUES ($1, $2, $3, $4)", p.member.Username, name, amount, start_date)
	if err != nil {
		log.Panic(err)
	}
}

func (p *Profile) Cancel_billing(name string) {
	_, err := p.b.db.Exec("UPDATE billing SET end_date = now() WHERE username = $1 AND name = $2 AND (end_date > now() OR end_date IS NULL)", p.member.Username, name)
	if err != nil {
		log.Panic(err)
	}
}

type Recurring_billing struct {
	Name       string
	Amount     float64
	Start_date time.Time
	End_date   pq.NullTime
}

func (p *Profile) Get_recurring_bills() (rb []Recurring_billing) {
	rows, err := p.b.db.Query("SELECT name, amount, start_date, end_date FROM billing WHERE username = $1 AND (end_date > now() OR end_date IS NULL)", p.member.Username)
	defer rows.Close()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return
	}
	for rows.Next() {
		var r Recurring_billing
		var amount string
		if err = rows.Scan(&r.Name, &amount, &r.Start_date, &r.End_date); err != nil {
			log.Panic(err)
		}
		if r.Amount, err = strconv.ParseFloat(amount[1:], 32); err != nil {
			log.Println(err)
		}
		rb = append(rb, r)
	}
	return
}

type Missed_payment struct {
	Name   string
	Amount float64
	Date   time.Time
}

func (p *Profile) Get_missed_payments() (mp []Missed_payment) {
	return
}

type Transaction struct {
	id         string
	username   string
	Date       time.Time
	Approved   bool
	Order_id   string
	Amount     float64
	Name       string // "Membership dues", "Storage fees", etc.
	Card       string // Last 4 digits
	Ip_address string
	billing_id int
}

func (p *Profile) New_transaction(amount float64, name, ip_address string) *Transaction {
	if amount <= 0 {
		return nil
	}
	order_id := fmt.Sprint(rand.Intn(1000000)) + "-" + p.member.Username
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
		log.Println("Payment of %.2f by %s failed", amount, p.member.Username)
	}
	txn := &Transaction{id: rsp.ID,
		username:   p.member.Username,
		Date:       time.Now(),
		Approved:   rsp.IsApproved(),
		Order_id:   rsp.OrderNumber,
		Amount:     amount,
		Name:       name,
		Card:       rsp.Card.LastFour,
		Ip_address: ip_address}
	_, err = p.b.db.Exec("INSERT INTO transaction (id, username, approved, order_id, amount, name, card, ip_address) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", rsp.ID, p.member.Username, txn.Approved, txn.Order_id, txn.Amount, txn.Name, txn.Card, txn.Ip_address)
	if err != nil {
		log.Panic(err)
	}
	return txn
}

func (p *Profile) Get_transactions(number int) []*Transaction {
	var txns []*Transaction
	rows, err := p.b.db.Query("SELECT id, approved, order_id, amount, name, card, ip_address, time FROM transaction WHERE username = $1 ORDER BY time DESC LIMIT $2", p.member.Username, number)
	defer rows.Close()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return txns
	}
	for i := 0; rows.Next(); i++ {
		txn := &Transaction{username: p.member.Username}
		txns = append(txns, txn)
		var (
			amount     string
			name       sql.NullString
			card       sql.NullString
			ip_address sql.NullString
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
