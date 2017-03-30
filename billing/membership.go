package billing

import (
	"database/sql"
	"log"
	"time"
)

//TODO: corporate memberships
func (p *Profile) New_pending_membership(is_student bool) *Invoice {
	var fee *Fee
	if is_student {
		fee = p.Find_fee("membership", "student")
	} else {
		fee = p.Find_fee("membership", "regular")
	}
	inv := p.New_recurring_bill(fee, p.member_id)
	if inv == nil {
		log.Panic("Invalid membership invoice for member ", p.member_id)
	}
	p.set_invoice_start_date(inv, time.Time{})
	return inv
}

func (p *Profile) Approve_pending_membership(i *Invoice) {
	if prorated := prorate_month(i.Fee.Amount);
		prorated > minimum_txn_amount {
		description := i.Fee.Description
		if prorated != i.Fee.Amount {
			description += " (prorated)"
		}
		first_inv := p.New_invoice(p.member_id, prorated, description, i.Fee)
		if txn := p.do_transaction(first_inv); txn == nil || !txn.Approved {
			p.do_missed_payment(first_inv, txn)
			//TODO: embed error
		}
	}
	p.set_invoice_start_date(i, time.Now())
}

// Get_membership also retrieves invoices pending approval
func (p *Profile) Get_membership() *Invoice {
	var id int
	if err := p.db.QueryRow(
		"SELECT i.id "+
			"FROM invoice i "+
			"JOIN fee f "+
			"ON f.id = i.fee "+
			"WHERE i.member = $1 "+
			"	AND f.category = 'membership'"+
			"	AND (i.end_date > now() OR i.end_date IS NULL)",
		p.member_id).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	return p.Billing.Get_bill(id)
}

//TODO: set end_date and regular membership invoice to start after
func (p *Profile) Change_to_student(grad_date time.Time) {
	invoice := p.Get_membership()
	date := invoice.Start_date
	p.Cancel_recurring_bill(invoice)
	invoice = p.New_recurring_bill(p.Find_fee("membership", "student"), p.member_id)
	p.set_invoice_start_date(invoice, date)
}

func (p *Profile) Change_from_student() {
	invoice := p.Get_membership()
	date := invoice.Start_date
	p.Cancel_recurring_bill(invoice)
	invoice = p.New_recurring_bill(p.Find_fee("membership", "regular"), p.member_id)
	p.set_invoice_start_date(invoice, date)
}

//TODO: Cancel storage and other makerspace-related invoices
//TODO: send card-cancellation e-mail to VITP
func (p *Profile) Cancel_membership() {
	invoice := p.Get_membership()
	p.Cancel_recurring_bill(invoice)
}
