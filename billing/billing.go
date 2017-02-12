package billing

import (
	"database/sql"
	beanstream "github.com/Beanstream/beanstream-go"
	"github.com/lib/pq"
	"github.com/vvanpo/makerspace/member"
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
	reports  beanstream.ReportsAPI
	Fees     map[string]Fee
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
	b.get_fees()
	go b.payment_scheduler()
	return b
}

type Fee struct {
	Id int
	Category string
	Identifier string
	Description string
	Amount float64
	Interval string
}

func (b *Billing) get_fees() {
	b.Fees = make(map[string]Fee)
	rows, err := b.db.Query("SELECT id, category, identifier, description, amount, recurring FROM fee")
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		f := Fee{}
		var amount sql.NullFloat64
		var interval sql.NullString
		if err = rows.Scan(&f.Id, &f.Category, &f.Identifier, &f.Description, &amount, &interval); err != nil {
			log.Panic(err)
		}
		f.Amount = amount.Float64
		f.Interval = interval.String
		b.Fees[f.Category + "." + f.Identifier] = f
	}
}

///TODO: allow members to register to pay for another member's fees (like their child)

// prorate_month returns the amount multiplied by the fraction of the current
//	month left.
func prorate_month(amount float64) float64 {
	days_in_month := first_of_next_month().AddDate(0, 0, -1).Day()
	days_left := days_in_month - time.Now().Day()
	return amount * float64(days_left) / float64(days_in_month)
}

type Invoice struct {
	Id          int
	Username    string
	Date        time.Time
	Amount      float64
	End_date    *time.Time
	Description string
	*Fee
	profile     *Profile
}

func (p *Profile) get_recurring_bills() {
	// Select recurring invoices without expired end-dates
	rows, err := p.db.Query("SELECT i.id, i.username, i.date, i.end_date, "+
		"COALESCE(i.description, f.description), COALESCE(i.amount, f.amount), "+
		"f.category, f.identifier, f.recurring FROM invoice i INNER JOIN fee f ON (i.fee = f.id) WHERE "+
		"i.profile = $1 AND f.recurring IS NOT NULL AND (i.end_date > now() OR "+
		"i.end_date IS NULL)",
		p.member.Username)
	defer rows.Close()
	if err != nil {
		if err == sql.ErrNoRows {
			return
		}
		log.Panic(err)
	}
	for rows.Next() {
		inv := &Invoice{profile: p}
		var end_date pq.NullTime
		if err = rows.Scan(&inv.Id, &inv.Username, &inv.Date, &end_date,
			&inv.Description, &inv.Amount, &inv.Category, &inv.Identifier, &inv.Interval); err != nil {
			log.Panic(err)
		}
		if end_date.Valid {
			inv.End_date = &end_date.Time
		}
		p.Invoices = append(p.Invoices, inv)
	}
}

//	Get_bill returns nil when the invoice isn't found.
func (p *Profile) Get_bill(id int) *Invoice {
	for _, i := range p.Invoices {
		if i.Id == id {
			return i
		}
	}
	return nil
}

func (p *Profile) New_recurring_bill(fee_id int, username string) {
	if member.Get(username, p.db) == nil {
		return
	}
	inv := &Invoice{
		Username: username,
		profile:  p}
	if err := p.db.QueryRow("INSERT INTO invoice (username, profile, fee) "+
		"VALUES ($1, $2, $3) RETURNING id, date, f.description, f.amount, "+
		"f.recurring FROM fee f ON (fee = f.id)", username, p.member.Username,
		fee_id).Scan(&inv.Id, &inv.Date, &inv.Description, &inv.Amount,
		&inv.Interval); err != nil {
		log.Panic(err)
	}
	p.Invoices = append(p.Invoices, inv)
}

func (i *Invoice) Cancel_recurring_bill() {
	for n, v := range i.profile.Invoices {
		if v == i {
			i.profile.Invoices = append(i.profile.Invoices[:n-1],
				i.profile.Invoices[n:]...)
		}
	}
	if _, err := i.profile.db.Exec("UPDATE invoice SET end_date = now() WHERE "+
		"id = $1 AND (end_date > now() OR end_date IS NULL)", i.Id);
		err != nil {
		log.Panic(err)
	}
	*i = Invoice{}
}

/*
	var id int
	var start_date time.Time
	if err := p.db.QueryRow("SELECT id, amount, start_date FROM billing WHERE"+
		"username = $1 AND name = $2 AND (end_date > now() OR end_date IS NULL)",
		p.member.Username, name).Scan(&id, &a, &start_date);
		err != == sql.ErrNoRows {

	}
	if err == sql.ErrNoRows {
		// Register billing
		_, err = p.db.Exec("INSERT INTO billing (username, name, amount) VALUES ($1, $2, $3)", p.member.Username, name, amount)
		if err != nil {
			log.Panic(err)
		}
		// Prorate the current month's bill, do transaction immediately.
		p.New_transaction(prorate_month(amount), name+" (prorated)", "")
		return
	} else if err != nil {
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

func (p *Profile) get_missed_payments() {
	return
}*/
