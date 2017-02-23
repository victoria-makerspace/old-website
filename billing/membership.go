package billing

import (
	"database/sql"
	"log"
	"time"
)

func (p *Profile) New_membership(is_student bool) *Invoice {
	var fee *Fee
	if is_student {
		fee = p.find_fee("membership", "student")
	} else {
		fee = p.find_fee("membership", "regular")
	}
	inv := p.New_recurring_bill(fee, p.member_id)
	prorated := prorate_month(fee.Amount)
	if txn := p.do_transaction(prorated, fee.Description+" (prorated)", inv);
		txn == nil {
		//TODO: missed payment, embed error
	} else if !txn.Approved {
		//TODO: missed payment, embed error
	}
	return inv
}

func (p *Profile) Get_membership() *Invoice {
	var id int
	if err := p.db.QueryRow(
		"SELECT i.id "+
		"FROM invoice i "+
		"JOIN fee f "+
		"ON f.id = i.fee "+
		"WHERE i.member = $1 "+
		"	AND f.category = 'membership'",
		p.member_id).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	return p.Billing.get_bill(id)
}

//TODO: set end_date and regular membership invoice to start after
func (p *Profile) Change_to_student(grad_date time.Time) {
	invoice := p.Get_membership()
	p.Cancel_recurring_bill(invoice)
	p.New_recurring_bill(p.find_fee("membership", "student"), p.member_id)
}

func (p *Profile) Change_from_student() {
	invoice := p.Get_membership()
	p.Cancel_recurring_bill(invoice)
	p.New_recurring_bill(p.find_fee("membership", "regular"), p.member_id)
}

//TODO: Cancel storage and other makerspace-related invoices
//TODO: send card-cancellation e-mail to VITP
func (p *Profile) Cancel_membership() {
	invoice := p.Get_membership()
	p.Cancel_recurring_bill(invoice)
}
