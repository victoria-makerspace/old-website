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
	Fees map[string]*Fee
	Storage
	db          *sql.DB
	config      beanstream.Config
	gateway     beanstream.Gateway
	payment_api beanstream.PaymentsAPI
	profile_api beanstream.ProfilesAPI
	report_api  beanstream.ReportsAPI
}

func Billing_new(merchant_id, payments_api_key, profiles_api_key, reports_api_key string, db *sql.DB) *Billing {
	rand.Seed(time.Now().UTC().UnixNano())
	b := &Billing{
		db:     db,
		config: beanstream.Config{merchant_id, payments_api_key, profiles_api_key, reports_api_key, "www", "api", "v1", "-8:00"}}
	b.gateway = beanstream.Gateway{b.config}
	b.payment_api = b.gateway.Payments()
	b.profile_api = b.gateway.Profiles()
	b.report_api = b.gateway.Reports()
	b.get_fees()
	b.get_storage()
	go b.payment_scheduler()
	return b
}

type Fee struct {
	Id          int
	Category    string
	Identifier  string
	Description string
	Amount      float64
	Interval    string
}

func (b *Billing) get_fees() {
	b.Fees = make(map[string]*Fee)
	rows, err := b.db.Query("SELECT id, category, identifier, description, amount, recurring FROM fee")
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		f := &Fee{}
		var amount sql.NullFloat64
		var interval sql.NullString
		if err = rows.Scan(&f.Id, &f.Category, &f.Identifier, &f.Description, &amount, &interval); err != nil {
			log.Panic(err)
		}
		f.Amount = amount.Float64
		f.Interval = interval.String
		b.Fees[f.Category+"_"+f.Identifier] = f
	}
}

func (b *Billing) get_fee(id int) *Fee {
	for _, f := range b.Fees {
		if f.Id == id {
			return f
		}
	}
	return nil
}

type Invoice struct {
	Id          int
	Username    string
	Date        time.Time
	Paid_by     *member.Member
	End_date    *time.Time
	Description string
	Amount      float64
	*Fee
	Interval string
}

func (b *Billing) get_bill(id int) *Invoice {
	inv := &Invoice{Id: id}
	var (
		paid_by                      sql.NullString
		end_date                     pq.NullTime
		description                  sql.NullString
		amount                       sql.NullFloat64
		interval                     sql.NullString
		fee_category, fee_identifier sql.NullString
	)
	if err := b.db.QueryRow("SELECT i.username, i.date, i.paid_by, "+
		"i.end_date, COALESCE(i.description, f.description), "+
		"COALESCE(i.amount, f.amount), COALESCE(i.recurring, f.recurring), "+
		"f.category, f.identifier, FROM invoice i LEFT JOIN fee f "+
		"ON (i.fee = f.id) WHERE i.id = $1", id).Scan(&inv.Username,
		&inv.Date, &paid_by, &end_date, &description, &amount, &interval,
		&fee_category, &fee_identifier); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	if m := member.Get(paid_by.String, b.db); m != nil {
		inv.Paid_by = m
	}
	if end_date.Valid {
		inv.End_date = &end_date.Time
	}
	inv.Description = description.String
	inv.Amount = amount.Float64
	inv.Interval = interval.String
	if fee_category.Valid && fee_identifier.Valid {
		inv.Fee = b.Fees[fee_category.String+"_"+fee_identifier.String]
	}
	return inv
}

func (b *Billing) get_bill_by_fee(fee *Fee, paid_by *member.Member) *Invoice {
	inv := &Invoice{Fee: fee, Paid_by: paid_by}
	var (
		end_date                     pq.NullTime
		description                  sql.NullString
		amount                       sql.NullFloat64
		interval                     sql.NullString
	)
	if err := b.db.QueryRow("SELECT i.id, i.username, i.date, "+
		"i.end_date, COALESCE(i.description, f.description), "+
		"COALESCE(i.amount, f.amount), COALESCE(i.recurring, f.recurring), "+
		"FROM invoice i JOIN fee f "+
		"ON (i.fee = $1) WHERE i.paid_by = $2", fee.Id, paid_by.Username).Scan(&inv.Id, &inv.Username,
		&inv.Date, &end_date, &description, &amount, &interval); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	if end_date.Valid {
		inv.End_date = &end_date.Time
	}
	inv.Description = description.String
	inv.Amount = amount.Float64
	inv.Interval = interval.String
	return inv
}

///TODO: allow members to register to pay for another member's fees (like their child)

func (p *Profile) get_recurring_bills() {
	// Select recurring invoices without expired end-dates
	rows, err := p.billing.db.Query("SELECT i.id, i.username, i.date, "+
		"i.end_date, COALESCE(i.description, f.description), "+
		"COALESCE(i.amount, f.amount), COALESCE(i.recurring, f.recurring), "+
		"f.category, f.identifier FROM invoice i LEFT JOIN fee f "+
		"ON (i.fee = f.id) WHERE CASE WHEN i.paid_by IS NULL THEN "+
		"i.username = $1 ELSE i.paid_by = $1 END AND "+
		"COALESCE(i.recurring, f.recurring) IS NOT NULL AND "+
		"(i.end_date > now() OR i.end_date IS NULL) ORDER BY i.date DESC",
		p.member.Username)
	defer rows.Close()
	if err != nil {
		if err == sql.ErrNoRows {
			return
		}
		log.Panic(err)
	}
	for rows.Next() {
		inv := &Invoice{Paid_by: p.member}
		var (
			end_date                     pq.NullTime
			description                  sql.NullString
			amount                       sql.NullFloat64
			interval                     sql.NullString
			fee_category, fee_identifier sql.NullString
		)
		if err = rows.Scan(&inv.Id, &inv.Username, &inv.Date, &end_date,
			&description, &amount, &interval, &fee_category, &fee_identifier); err != nil {
			log.Panic(err)
		}
		if end_date.Valid {
			inv.End_date = &end_date.Time
		}
		inv.Description = description.String
		inv.Amount = amount.Float64
		inv.Interval = interval.String
		if fee_category.Valid && fee_identifier.Valid {
			inv.Fee = p.billing.Fees[fee_category.String+"_"+
				fee_identifier.String]
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

//TODO: BUG: not all 'fee' records have non-null 'recurring' fields
func (p *Profile) New_recurring_bill(fee_id int, username string) {
	fee := p.billing.get_fee(fee_id)
	inv := &Invoice{Username: username,
		Paid_by:     p.member,
		Description: fee.Description,
		Amount:      fee.Amount,
		Interval:    fee.Interval,
		Fee:         fee}
	if username != p.member.Username && member.Get(username, p.billing.db) == nil {
		return
	}
	if err := p.billing.db.QueryRow("INSERT INTO invoice (username, paid_by, "+
		"fee) VALUES ($1, $2, $3) RETURNING id, date", username,
		p.member.Username, fee_id).Scan(&inv.Id, &inv.Date); err != nil {
		log.Panic(err)
	}
	p.Invoices = append(p.Invoices, inv)
}

func (p *Profile) Cancel_recurring_bill(i *Invoice) {
	if i == nil || i.Paid_by != p.member {
		return
	}
	for n, v := range p.Invoices {
		if v == i {
			p.Invoices = append(p.Invoices[:n], p.Invoices[n+1:]...)
		}
	}
	if _, err := p.billing.db.Exec("UPDATE invoice SET end_date = now() WHERE "+
		"id = $1 AND (end_date > now() OR end_date IS NULL)", i.Id); err != nil {
		log.Panic(err)
	}
	*i = Invoice{}
}

func (p *Profile) get_missed_payments() {
	return
}

// prorate_month returns the amount multiplied by the fraction of the current
//	month left.
func prorate_month(amount float64) float64 {
	days_in_month := first_of_next_month().AddDate(0, 0, -1).Day()
	days_left := days_in_month - time.Now().Day()
	return amount * float64(days_left) / float64(days_in_month)
}
