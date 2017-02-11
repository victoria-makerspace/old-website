package billing

import (
	"database/sql"
	beanstream "github.com/Beanstream/beanstream-go"
	"github.com/lib/pq"
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


// prorate_month returns the amount multiplied by the fraction of the current
//	month left.
func prorate_month(amount float64) float64 {
	days_in_month := first_of_next_month().AddDate(0, 0, -1).Day()
	days_left := days_in_month - time.Now().Day()
	return amount * float64(days_left) / float64(days_in_month)
}

func (p *Profile) Update_billing(name string, amount float64) {
	var id int
	var a string
	var start_date time.Time
	err := p.db.QueryRow("SELECT id, amount, start_date FROM billing WHERE username = $1 AND name = $2 AND (end_date > now() OR end_date IS NULL)", p.member.Username, name).Scan(&id, &a, &start_date)
	if err == sql.ErrNoRows {
		// Register billing
		_, err = p.db.Exec("INSERT INTO billing (username, name, amount) VALUES ($1, $2, $3)", p.member.Username, name, amount)
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
	_, err = p.db.Exec("INSERT INTO billing (username, name, amount, start_date) VALUES ($1, $2, $3, $4)", p.member.Username, name, amount, start_date)
	if err != nil {
		log.Panic(err)
	}
}

func (p *Profile) Cancel_billing(name string) {
	_, err := p.db.Exec("UPDATE billing SET end_date = now() WHERE username = $1 AND name = $2 AND (end_date > now() OR end_date IS NULL)", p.member.Username, name)
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
	rows, err := p.db.Query("SELECT name, amount, start_date, end_date FROM billing WHERE username = $1 AND (end_date > now() OR end_date IS NULL)", p.member.Username)
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

// Update_membership should only be called after student information has already
//	been updated.
func (bp *Profile) Update_membership() {
	if 
	var query string
	if bp.Student != nil {
		query = ""
	}
}

type Missed_payment struct {
	Name   string
	Amount float64
	Date   time.Time
}

func (p *Profile) Get_missed_payments() (mp []Missed_payment) {
	return
}

