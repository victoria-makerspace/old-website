package billing

import (
	"time"
)

func (p *Profile) New_membership() {
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
	p.do_transaction(prorated, fee.Description+" (prorated)", inv)
}

func (p *Profile) Get_membership() *Invoice {
	for _, i := range p.Invoices {
		if i.Fee.Category == "membership" {
			return i
		}
	}
	if i := p.billing.get_bill_by_fee(p.billing.Fees["membership_regular"],
		p.member); i != nil {
		return i
	} else if i := p.billing.get_bill_by_fee(
		p.billing.Fees["membership_regular"], p.member); i != nil {
		return i
	}
	return p.billing.get_bill_by_fee(p.billing.Fees["membership_corporate"],
		p.member)
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
