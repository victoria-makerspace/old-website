package billing

import (
	"time"
)

func (p *Profile) New_membership() {
	//TODO: change member.Active to current membership Invoice object
	if p.member.Active {
		return
	}
	member_type := "membership_regular"
	if p.member.Student {
		member_type = "membership_student"
	} else if p.member.Corporate {
		member_type = "membership_corporate"
		//TODO
	}
	fee := p.billing.Fees[member_type]
	inv := p.New_recurring_bill(fee.Id, p.member.Username)
	prorated := prorate_month(fee.Amount)
	if txn := p.do_transaction(prorated, fee.Description+" (prorated)", inv);
		txn == nil {
		if !txn.Approved {
			//TODO: missed payment, embed error
		}
	}
}

func (p *Profile) Get_membership() *Invoice {
	for _, i := range p.Invoices {
		if i.Fee.Category == "membership" {
			return i
		}
	}
	//TODO: return invoice when membership paid by someone else
	return nil
}

func (p *Profile) Update_student(institution, email string, grad_date time.Time) {
	invoice := p.Get_membership()
	was_student := p.member.Student
	p.member.Update_student(institution, email, grad_date)
	if !was_student && invoice != nil {
		p.Cancel_recurring_bill(invoice)
		p.New_recurring_bill(p.billing.Fees["membership_student"].Id,
			p.member.Username)
	}
}

func (p *Profile) Delete_student() {
	invoice := p.Get_membership()
	was_student := p.member.Student
	p.member.Delete_student()
	if was_student && invoice != nil {
		p.Cancel_recurring_bill(invoice)
		p.New_recurring_bill(p.billing.Fees["membership_regular"].Id,
			p.member.Username)
	}
}

//TODO: Cancel storage and other makerspace-related invoices
//TODO: send card-cancellation e-mail to VITP
func (p *Profile) Cancel_membership() {
	invoice := p.Get_membership()
	p.Cancel_recurring_bill(invoice)
	p.member.Active = false
}
