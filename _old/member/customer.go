package member

import (
	"fmt"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/card"
	"github.com/stripe/stripe-go/customer"
	"log"
	"strings"
)

type Customer struct {
	*stripe.Customer
	// key == subscription ID
	Subscriptions map[string]*stripe.Sub
}

//TODO: be explicit throughout member package about calls to Get_customer(),
//	as they involve a cross-origin request
func (m *Member) Get_customer() *Customer {
	if m.Customer_id != "" && m.customer == nil {
		c, err := customer.Get(m.Customer_id, nil)
		if err != nil {
			return nil
		}
		if !c.Deleted {
			m.customer = &Customer{Customer: c}
			m.get_subscriptions()
		}
	}
	return m.customer
}

func (m *Member) Update_customer(token string) error {
	params := &stripe.CustomerParams{
		Desc:  m.Name + "'s account",
		Email: m.Email}
	params.Meta = map[string]string{"member_id": fmt.Sprint(m.Id)}
	if token != "" {
		params.SetSource(token)
	}
	if m.Customer_id == "" {
		cust, err := customer.New(params)
		if err != nil {
			return err
		}
		if _, err := m.Exec(
			"UPDATE member "+
				"SET stripe_customer_id = $2 "+
				"WHERE id = $1", m.Id, cust.ID); err != nil {
			log.Panic(err)
		}
		m.Customer_id = cust.ID
		m.customer = &Customer{Customer: cust, Subscriptions: make(map[string]*stripe.Sub)}
		return nil
	}
	cust, err := customer.Update(m.Customer_id, params)
	if err != nil {
		return err
	}
	m.customer = &Customer{Customer: cust}
	m.get_subscriptions()
	return nil
}

func (m *Member) Get_payment_source() *stripe.PaymentSource {
	if c := m.Get_customer(); c != nil {
		return c.DefaultSource
	}
	return nil
}

func (m *Member) Get_card() *stripe.Card {
	if cust := m.Get_customer(); cust != nil && cust.DefaultSource != nil &&
		strings.HasPrefix(cust.DefaultSource.ID, "card") {
		c, _ := card.Get(cust.DefaultSource.ID, &stripe.CardParams{
			Customer: cust.ID})
		return c
	}
	return nil
}
