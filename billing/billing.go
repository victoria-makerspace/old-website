package billing

import (
	"database/sql"
	beanstream "github.com/Beanstream/beanstream-go"
	"github.com/lib/pq"
	"log"
	"math/rand"
	"time"
)

type Billing struct {
	Fees        map[int]*Fee
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
		db: db,
		config: beanstream.Config{
			merchant_id,
			payments_api_key,
			profiles_api_key,
			reports_api_key,
			"www", "api", "v1", "-8:00"}}
	b.gateway = beanstream.Gateway{b.config}
	b.payment_api = b.gateway.Payments()
	b.profile_api = b.gateway.Profiles()
	b.report_api = b.gateway.Reports()
	b.get_fees()
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
	b.Fees = make(map[int]*Fee)
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
		b.Fees[f.Id] = f
	}
}

func (b *Billing) Find_fee(category, identifier string) *Fee {
	for _, f := range b.Fees {
		if f.Category == category && f.Identifier == identifier {
			return f
		}
	}
	return nil
}

type Invoice struct {
	Id          int
	Member      int
	Paid_by     int
	Start_date  time.Time
	End_date    time.Time
	Description string
	Amount      float64
	*Fee
	Interval string
}

func (b *Billing) Get_bill(id int) *Invoice {
	inv := &Invoice{Id: id}
	var (
		start_date	pq.NullTime
		end_date	pq.NullTime
		description sql.NullString
		interval    sql.NullString
		fee_id      sql.NullInt64
	)
	if err := b.db.QueryRow("SELECT i.member, i.start_date, i.paid_by, "+
		"i.end_date, COALESCE(i.description, f.description), "+
		"COALESCE(i.amount, f.amount), COALESCE(i.recurring, f.recurring), "+
		"f.id FROM invoice i LEFT JOIN fee f "+
		"ON i.fee = f.id WHERE i.id = $1", id).Scan(&inv.Member,
		&start_date, &inv.Paid_by, &end_date, &description, &inv.Amount,
		&interval, &fee_id); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	inv.Start_date = start_date.Time
	inv.End_date = end_date.Time
	inv.Description = description.String
	inv.Interval = interval.String
	if fee_id.Valid {
		inv.Fee = b.Fees[int(fee_id.Int64)]
	}
	return inv
}

//TODO: return a slice
func (b *Billing) get_bill_by_fee(fee *Fee, paid_by int) *Invoice {
	inv := &Invoice{Fee: fee, Paid_by: paid_by}
	var (
		start_date    pq.NullTime
		end_date    pq.NullTime
		description sql.NullString
		interval    sql.NullString
	)
	if err := b.db.QueryRow("SELECT i.id, i.member, i.start_date, "+
		"i.end_date, COALESCE(i.description, f.description), "+
		"COALESCE(i.amount, f.amount), COALESCE(i.recurring, f.recurring) "+
		"FROM invoice i JOIN fee f "+
		"ON i.fee = $1 WHERE i.paid_by = $2", fee.Id, paid_by).Scan(&inv.Id,
		&inv.Member, &start_date, &end_date, &description, &inv.Amount,
		&interval); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	inv.Start_date = start_date.Time
	inv.End_date = end_date.Time
	inv.Description = description.String
	inv.Interval = interval.String
	return inv
}

/*
//TODO: useless, always get bills via profile
func (b *Billing) get_all_recurring(interval string) []*Invoice {
	inv := make([]*Invoice, 0)
	rows, err := b.db.Query(
		"SELECT "+
		"	i.id, i.username, i.date, "+
		"	COALESCE(i.paid_by, i.username) "+
		"	i.end_date, "+
		"	COALESCE(i.description, f.description), "+
		"	COALESCE(i.amount, f.amount), "+
		"	i.fee "+
		"FROM invoice i "+
		"LEFT JOIN fee f "+
		"ON (i.fee = f.id) "+
		"WHERE "+
		"	COALESCE(i.recurring, f.recurring) = $1 "+
		"	AND (i.end_date > now() OR i.end_date IS NULL)",
		interval)
	defer rows.Close()
	if err != nil {
		if err == sql.ErrNoRows {
			return inv
		}
		log.Panic(err)
	}
	for rows.Next() {
		i := &Invoice{Interval: interval}
		var (
			end_date pq.NullTime
			description sql.NullString
			fee_id sql.NullInt64
		)
		if err := rows.Scan(&i.Id, &i.Username, &i.Paid_by, &end_date, &description, &i.Amount, &fee_id); err != nil {
			log.Panic(err)
		}
		if end_date.Valid {
			i.End_date = &end_date.Time
		}
		i.Description = description.String
		i.Fee = b.get_fee(int(fee_id.Int64))
		inv = append(inv, i)
	}
	return inv
}*/

//TODO: break out invoice methods into invoice.go
///TODO: allow members to register to pay for another member's fees (like their child)

func (p *Profile) get_recurring_bills() {
	// Select recurring invoices not pending approval without expired end-dates
	rows, err := p.db.Query(
		"SELECT "+
			"	i.id, i.member, i.start_date, i.end_date, "+
			"	COALESCE(i.description, f.description), "+
			"	COALESCE(i.amount, f.amount), "+
			"	COALESCE(i.recurring, f.recurring), "+
			"	i.fee "+
			"FROM invoice i "+
			"LEFT JOIN fee f "+
			"ON i.fee = f.id "+
			"WHERE "+
			"	i.paid_by = $1 "+
			"	AND start_date IS NOT NULL "+
			"	AND COALESCE(i.recurring, f.recurring) IS NOT NULL "+
			"	AND (i.end_date > now() OR i.end_date IS NULL) "+
			"ORDER BY i.start_date DESC",
		p.member_id)
	defer rows.Close()
	if err != nil {
		if err == sql.ErrNoRows {
			return
		}
		log.Panic(err)
	}
	for rows.Next() {
		inv := &Invoice{Paid_by: p.member_id}
		var (
			end_date    pq.NullTime
			description sql.NullString
			fee_id      sql.NullInt64
		)
		if err = rows.Scan(&inv.Id, &inv.Member, &inv.Start_date, &end_date,
			&description, &inv.Amount, &inv.Interval, &fee_id); err != nil {
			log.Panic(err)
		}
		inv.End_date = end_date.Time
		inv.Description = description.String
		inv.Fee = p.Fees[int(fee_id.Int64)]
		p.Invoices = append(p.Invoices, inv)
	}
}

func (p *Profile) New_invoice(member_id int, amount float64, description string, fee *Fee) *Invoice {
	if amount == 0 {
		if fee == nil || fee.Amount == 0 {
			log.Panic("Invalid invoice (amount cannot be 0)")
		}
		amount = fee.Amount
	}
	if amount < minimum_txn_amount {
		log.Printf("Invoice for member %d below minimum amount ($%0.2f < $%0.2f)",
			p.member_id, amount, minimum_txn_amount)
		return nil
	}
	if description == "" && fee != nil {
		description = fee.Description
	} else {
		log.Panic("Empty description for new invoice")
	}
	inv := &Invoice{Member: member_id,
		Paid_by:     p.member_id,
		Description: description,
		Amount:      amount,
		Fee:         fee}
	if fee != nil {
		if err := p.db.QueryRow(
			"INSERT INTO invoice ("+
				"	member, paid_by, end_date, description, amount, fee"+
				") "+
				"VALUES ($1, $2, 'epoch', $3, $4, $5) "+
				"RETURNING id, start_date, end_date",
			member_id, p.member_id, inv.Description, amount,
			fee.Id).Scan(&inv.Id, &inv.Start_date, &inv.End_date); err != nil {
			log.Panic(err)
		}
	} else {
		if err := p.db.QueryRow(
			"INSERT INTO invoice ("+
				"	member, paid_by, end_date, description, amount"+
				") "+
				"VALUES ($1, $2, 'epoch', $3, $4) "+
				"RETURNING id, start_date, end_date",
			member_id, p.member_id, inv.Description, amount).Scan(&inv.Id,
			&inv.Start_date, &inv.End_date); err != nil {
			log.Panic(err)
		}
	}
	return inv
}

//TODO: BUG: not all 'fee' records have non-null 'recurring' fields
func (p *Profile) New_recurring_bill(fee *Fee, member_id int) *Invoice {
	if fee == nil {
		log.Panic("Nil fee argument");
	}
	if fee.Amount < minimum_txn_amount {
		log.Printf("Invoice for member %d below minimum amount ($%0.2f < $%0.2f)",
			p.member_id, fee.Amount, minimum_txn_amount)
		return nil
	}
	inv := &Invoice{Member: member_id,
		Paid_by:     p.member_id,
		Description: fee.Description,
		Amount:      fee.Amount,
		Interval:    fee.Interval,
		Fee:         fee}
	if err := p.db.QueryRow("INSERT INTO invoice (member, paid_by, "+
		"fee) VALUES ($1, $2, $3) RETURNING id, start_date", member_id,
		p.member_id, fee.Id).Scan(&inv.Id, &inv.Start_date); err != nil {
		log.Panic(err)
	}
	p.Invoices = append(p.Invoices, inv)
	return inv
}

func (p *Profile) Cancel_recurring_bill(i *Invoice) {
	if i == nil || i.Paid_by != p.member_id {
		log.Panic("Invalid invoice")
	}
	for n, v := range p.Invoices {
		if v == i {
			p.Invoices = append(p.Invoices[:n], p.Invoices[n+1:]...)
		}
	}
	query := "UPDATE invoice SET end_date = now() WHERE "+
		"id = $1 AND (end_date > now() OR end_date IS NULL)"
	if i.Start_date.IsZero() {
		query = "DELETE FROM invoice WHERE id = $1"
	}
	if _, err := p.db.Exec(query, i.Id); err != nil {
		log.Panic(err)
	}
	if i.Fee != nil && i.Fee.Category == "storage" {
		if _, err := p.db.Exec(
			"UPDATE storage "+
				"SET invoice = NULL "+
				"WHERE invoice = $1", i.Id); err != nil {
			log.Panic(err)
		}
	}
	*i = Invoice{}
}

func (b *Billing) set_invoice_start_date(i *Invoice, date time.Time) {
	if !i.End_date.IsZero() && date.After(i.End_date) {
		log.Panic("Invalid start date for invoice ", i.Id)
	}
	if date.IsZero() {
		if _, err := b.db.Exec(
			"UPDATE invoice "+
				"SET start_date = NULL "+
				"WHERE id = $1", i.Id); err != nil {
			log.Panic(err)
		}
	} else {
		if _, err := b.db.Exec(
			"UPDATE invoice "+
				"SET start_date = $2 "+
				"WHERE id = $1", i.Id, date); err != nil {
			log.Panic(err)
		}
	}
	i.Start_date = date
}

// prorate_month returns the amount multiplied by the fraction of the current
//	month left.
func prorate_month(amount float64) float64 {
	days_in_month := first_of_next_month().AddDate(0, 0, -1).Day()
	days_left := days_in_month - time.Now().Day()
	return amount * float64(days_left) / float64(days_in_month)
}
