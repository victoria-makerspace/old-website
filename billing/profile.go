package billing

import (
	"database/sql"
	beanstream "github.com/Beanstream/beanstream-go"
	"github.com/vvanpo/makerspace/member"
	"log"
)

type Profile struct {
	member *member.Member
	*Billing
	beanstream.Profile
	Error    *string
	Invoices []*Invoice
}

func (b *Billing) New_profile(token, cardholder string, m *member.Member) *Profile {
	p := &Profile{Billing: b, member: m}
	p.Token = beanstream.Token{
		Token: token,
		Name:  cardholder}
	p.Custom = beanstream.CustomFields{Ref1: m.Username}
	rsp, err := b.profiles.CreateProfile(p.Profile)
	if err != nil {
		log.Println("Failed to create profile: ", err)
		return nil
	}
	p.Id = rsp.Id
	if _, err = b.db.Exec("INSERT INTO payment_profile VALUES ($1, $2)",
		m.Username, rsp.Id); err != nil {
		log.Panic(err)
	}
	return p
}

func (b *Billing) Get_profile(m *member.Member) *Profile {
	var (
		id            string
		invalid       sql.NullString
	)
	err := b.db.QueryRow("SELECT id, invalid_error FROM payment_profile"+
		" WHERE username = $1", m.Username).Scan(&id, &invalid)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return nil
	}
	p := &Profile{Billing: b, member: m}
	if bs, err := b.profiles.GetProfile(id); err != nil {
		log.Println(err)
		return nil
	} else {
		p.Profile = *bs
		if invalid.Valid {
			p.Error = &invalid.String
		}
	}
	p.get_recurring_bills()
	return p
}

func (p *Profile) Get_card() *beanstream.CreditCard {
	if p.Card.Number == "" {
		return nil
	}
	return &p.Card
}

func (p *Profile) Delete_card() {
	if _, err := p.DeleteCard(p.profiles, 1); err != nil {
		log.Println(err)
	}
	p.Card = beanstream.CreditCard{}
}

func (p *Profile) Update_card(name, token string) {
	if p.Get_card() != nil {
		p.Delete_card()
	}
	if _, err := p.profiles.AddTokenizedCard(p.Id, name, token); err != nil {
		log.Println(err)
		return
	}
	card, err := p.Profile.GetCard(p.profiles, 1)
	if err != nil {
		log.Println(err)
		return
	}
	// Clear card error
	if _, err = p.db.Exec("UPDATE payment_profile SET error = false, error_message = null"); err != nil {
		log.Panic(err)
	}
	p.Card = *card
}
