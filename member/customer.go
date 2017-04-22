package member

import (
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"
	"log"
	"fmt"
)

type Customer struct {
	*stripe.Customer
	Subscriptions map[string]*stripe.Sub
}

func (m *Member) Customer() *Customer {
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
		cust, err = customer.Update(m.Customer_id, params)
	}
	if err != nil {
		return fmt.Errorf(err.(*stripe.Error).Msg)
	}
	if m.Customer_id == "" {
		if _, err := m.Exec(
			"UPDATE member "+
				"SET stripe_customer_id = $2 "+
				"WHERE id = $1", m.Id, cust.ID); err != nil {
			log.Panic(err)
		}
		m.Customer_id = cust.ID
		m.customer = &Customer{Customer: cust}
	} else {
		m.customer.Customer = cust
		m.get_subscriptions()
	}
	return nil
}

func (m *Member) Has_card() bool {
	if c := m.Customer(); c != nil && c.DefaultSource != nil {
		return true
	}
	return false
}
