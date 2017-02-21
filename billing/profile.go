package billing

import (
	"database/sql"
	beanstream "github.com/Beanstream/beanstream-go"
	"github.com/vvanpo/makerspace/member"
	"log"
)

//TODO: use the actual Error interface
type Profile struct {
	Invoices     []*Invoice
	Transactions []*Transaction
	Error        *string
	billing      *Billing
	bs_profile   *beanstream.Profile
	member       *member.Member
}

func (b *Billing) Get_profile(m *member.Member) *Profile {
	p := &Profile{billing: b, member: m}
	var (
		id      string
		invalid sql.NullString
	)
	err := b.db.QueryRow("SELECT id, invalid_error FROM payment_profile "+
		"WHERE username = $1", m.Username).Scan(&id, &invalid)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		*p.Error = "No credit card profile"
		return p
	}
	if bs, err := b.profile_api.GetProfile(id); err != nil {
		log.Println(err)
		p.set_error("No credit card profile")
		return p
	} else {
		p.bs_profile = bs
		if invalid.Valid {
			p.Error = &invalid.String
		}
	}
	p.get_recurring_bills()
	p.get_transactions()
	return p
}

func (p *Profile) Get_card() *beanstream.CreditCard {
	if p.bs_profile == nil || p.bs_profile.Card.Number == "" {
		return nil
	}
	return &p.bs_profile.Card
}

func (p *Profile) Delete_card() {
	if p.Get_card() == nil {
		return
	}
	if _, err := p.bs_profile.DeleteCard(p.billing.profile_api, 1); err != nil {
		log.Println(err)
	}
	p.set_error("no card")
	p.bs_profile.Card = beanstream.CreditCard{}
}

func (p *Profile) Update_card(token, cardholder string) {
	if p.Get_card() != nil {
		p.Delete_card()
	}
	if p.bs_profile == nil {
		p.new_bs_profile(token, cardholder)
		return
	}
	if _, err := p.billing.profile_api.AddTokenizedCard(p.bs_profile.Id,
		cardholder, token); err != nil {
		log.Println(err)
		return
	}
	card, err := p.bs_profile.GetCard(p.billing.profile_api, 1)
	if err != nil {
		log.Println(err)
		return
	}
	// Clear card error
	p.clear_error()
	p.bs_profile.Card = *card
}

func (p *Profile) new_bs_profile(token, cardholder string) {
	p.bs_profile.Token = beanstream.Token{
		Token: token,
		Name:  cardholder}
	p.bs_profile.Custom = beanstream.CustomFields{Ref1: p.member.Username}
	rsp, err := p.billing.profile_api.CreateProfile(*p.bs_profile)
	if err != nil {
		log.Println("Failed to create profile: ", err)
		return
	}
	p.bs_profile.Id = rsp.Id
	if _, err = p.billing.db.Exec("INSERT INTO payment_profile VALUES ($1, $2)",
		p.member.Username, rsp.Id); err != nil {
		log.Panic(err)
	}
	p.clear_error()
}

func (p *Profile) set_error(err string) {
	*p.Error = err
	if _, e := p.billing.db.Exec("UPDATE payment_profile "+
		"SET invalid_error = $1", err); e != nil {
		log.Panic(e)
	}
}

func (p *Profile) clear_error() {
	p.Error = nil
	if _, err := p.billing.db.Exec("UPDATE payment_profile " +
		"SET invalid_error = NULL"); err != nil {
		log.Panic(err)
	}
}
