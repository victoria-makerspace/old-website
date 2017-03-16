package billing

import (
	"database/sql"
	"fmt"
	beanstream "github.com/Beanstream/beanstream-go"
	"log"
)

//TODO: use the actual Error interface
type Error int

const (
	None Error = iota
	No_profile
	No_card
	Invalid_card
	Expired_card
)

type Profile struct {
	Invoices []*Invoice
	Error
	*Billing
	Missed_payments []*Missed_payment
	bs_id           string
	bs_profile      *beanstream.Profile
	member_id       int
}

func (b *Billing) New_profile(member_id int) *Profile {
	if _, err := b.db.Exec(
		"INSERT INTO payment_profile (member) "+
			"VALUES ($1)",
		member_id); err != nil {
		log.Panic(err)
	}
	return &Profile{Billing: b, Error: No_profile, member_id: member_id}
}

func (b *Billing) Get_profile(member_id int) *Profile {
	p := &Profile{Billing: b, member_id: member_id}
	var (
		profile_id sql.NullString
		invalid    sql.NullInt64
	)
	err := b.db.QueryRow("SELECT id, error FROM payment_profile "+
		"WHERE member = $1", member_id).Scan(&profile_id, &invalid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	p.bs_id = profile_id.String
	p.Error = Error(invalid.Int64)
	/////TODO: p.Recurring_bills() if len(Invoices) = 0, get_recurring...etc
	p.get_recurring_bills()
	return p
}

func (p *Profile) get_bs_profile() *beanstream.Profile {
	if p.bs_id == "" {
		return nil
	}
	if p.bs_profile != nil {
		return p.bs_profile
	}
	bs, err := p.profile_api.GetProfile(p.bs_id)
	if err != nil {
		log.Println(err)
		p.set_error(No_profile)
		return nil
	}
	p.bs_profile = bs
	return bs
}

func (p *Profile) Get_card() *beanstream.CreditCard {
	if p.get_bs_profile() == nil || p.bs_profile.Card.Number == "" {
		return nil
	}
	return &p.bs_profile.Card
}

func (p *Profile) Delete_card() {
	if p.Get_card() == nil {
		return
	}
	if _, err := p.bs_profile.DeleteCard(p.profile_api, 1); err != nil {
		log.Println(err)
	}
	p.set_error(No_card)
	p.bs_profile.Card = beanstream.CreditCard{}
}

func (p *Profile) Update_card(token, cardholder string) {
	if p.Get_card() != nil {
		p.Delete_card()
	} else if p.get_bs_profile() == nil {
		p.new_bs_profile(token, cardholder)
		return
	}
	if _, err := p.profile_api.AddTokenizedCard(p.bs_profile.Id, cardholder,
		token); err != nil {
		log.Println(err)
		return
	}
	card, err := p.bs_profile.GetCard(p.profile_api, 1)
	if err != nil {
		log.Println(err)
		return
	}
	// Clear card error
	p.clear_error()
	p.bs_profile.Card = *card
	p.Retry_missed_payments()
}

func (p *Profile) new_bs_profile(token, cardholder string) {
	p.bs_profile = &beanstream.Profile{
		Token: beanstream.Token{
			Token: token,
			Name:  cardholder}}
	p.bs_profile.Custom = beanstream.CustomFields{Ref1: fmt.Sprint(p.member_id)}
	rsp, err := p.profile_api.CreateProfile(*p.bs_profile)
	if err != nil {
		log.Println("Failed to create profile: ", err)
		return
	}
	p.bs_id = rsp.Id
	p.bs_profile.Id = rsp.Id
	if _, err = p.db.Exec(
		"UPDATE payment_profile "+
			"SET id = $2 "+
			"WHERE member = $1",
		p.member_id, rsp.Id); err != nil {
		log.Panic(err)
	}
	p.clear_error()
}

func (p *Profile) set_error(err Error) {
	p.Error = err
	if _, e := p.db.Exec("UPDATE payment_profile "+
		"SET error = $1 "+
		"WHERE member = $2", err, p.member_id); e != nil {
		log.Panic(e)
	}
}

func (p *Profile) clear_error() {
	p.Error = None
	if _, err := p.db.Exec("UPDATE payment_profile "+
		"SET error = NULL "+
		"WHERE member = $1", p.member_id); err != nil {
		log.Panic(err)
	}
}

// Retrieves all member profiles with active invoices
func (b *Billing) get_all_profiles() []*Profile {
	profiles := make([]*Profile, 0)
	rows, err := b.db.Query("SELECT member FROM payment_profile")
	defer rows.Close()
	if err != nil {
		if err == sql.ErrNoRows {
			return profiles
		}
		log.Panic(err)
	}
	for rows.Next() {
		var member_id int
		if err := rows.Scan(&member_id); err != nil {
			log.Panic(err)
		}
		profiles = append(profiles, b.Get_profile(member_id))
	}
	return profiles
}
