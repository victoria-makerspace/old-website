package member

import (
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"
	"log"
	"fmt"
)

func (m *Member) Customer() *stripe.Customer {
	if m.Customer_id != "" && m.customer == nil {
		c, err := customer.Get(m.Customer_id, nil)
		if err != nil {
			return nil
		}
		if !c.Deleted {
			m.customer = c
		}
	}
	return m.customer
}

func (m *Member) Update_customer(token string, params *stripe.CustomerParams) error {
	if params == nil {
		params = &stripe.CustomerParams{}
	}
	if token != "" {
		params.SetSource(token)
	}
	params.Desc = m.Name + "'s account"
	params.Email = m.Email
	var err error
	var cust *stripe.Customer
	if m.Customer_id == "" {
		cust, err = customer.New(params)
	} else {
		cust, err = customer.Update(m.customer.ID, params)
	}
	if err != nil {
		return fmt.Errorf(err.(*stripe.Error).Msg)
	}
	m.customer = cust
	if m.Customer_id == "" {
		m.Customer_id = m.customer.ID
		if _, err := m.Exec(
			"UPDATE member "+
				"SET stripe_customer_id = $2 "+
				"WHERE id = $1", m.Id, m.customer.ID); err != nil {
			log.Panic(err)
		}
	}
	return nil
}

func (m *Member) Has_card() bool {
	if c := m.Customer(); c != nil && c.DefaultSource != nil {
		return true
	}
	return false
}

func (m *Member) Active_subscriptions() map[string]*stripe.Sub {
	subs := make(map[string]*stripe.Sub)
	if m.Customer() != nil {
		for _, s := range m.customer.Subs.Values {
			if s.Ended == 0 {
				subs[s.ID] = s
			}
		}
	}
	return subs
}
