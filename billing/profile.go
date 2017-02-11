package billing

import (
	"database/sql"
	beanstream "github.com/Beanstream/beanstream-go"
	"github.com/lib/pq"
	"github.com/vvanpo/makerspace/member"
	"log"
)

type Profile struct {
	member	 *member.Member
	*Billing
	beanstream.Profile
	Student  *student
}

func (b *Billing) New_profile(token, cardholder string, m *member.Member) *Profile {
	p := &Profile{Billing: b, member: m}
	p.Token = beanstream.Token{
		Token: token,
		Name:  cardholder}
	p.Custom = beanstream.CustomFields{Ref1: m.Username}
	rsp, err := b.profiles.CreateProfile(p.Profile)
	if err != nil {
		log.Println(err)
	}
	p.Id = rsp.Id
	if _, err = b.db.Exec("INSERT INTO billing_profile VALUES ($1, $2)", m.Username, rsp.Id); err != nil {
		log.Panic(err)
	}
	return p
}

func (b *Billing) Get_profile(m *member.Member) *Profile {
	var (
		id string
		username, institution sql.NullString
		grad_date pq.NullTime
	)
	err := b.db.QueryRow("SELECT bp.id, s.username, s.institution, s.graduation_date FROM billing_profile bp LEFT JOIN student s USING (username) WHERE bp.username = $1", m.Username).Scan(&id, &username, &institution, &grad_date)
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
		if username.Valid {
			p.Student = &student{institution.String, grad_date.Time}
		}
	}
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
	p.Card = *card
}
