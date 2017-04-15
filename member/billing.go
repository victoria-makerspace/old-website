package member

import (
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"
	"github.com/stripe/stripe-go/plan"
	"log"
	"fmt"
	"strconv"
	"time"
)

func (m *Member) Customer() *stripe.Customer {
	if m.customer_id != "" && m.customer == nil {
		c, err := customer.Get(m.customer_id, nil)
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
	if m.customer_id == "" {
		if m.customer, err = customer.New(params); err == nil {
			m.customer_id = m.customer.ID
		}
	} else {
		m.customer, err = customer.Update(m.customer.ID, params)
	}
	if err != nil {
		return err
	}
	if _, err := m.Exec(
		"UPDATE member "+
			"SET stripe_customer_id = $2 "+
			"WHERE id = $1", m.Id, m.customer.ID); err != nil {
		log.Panic(err)
	}
	return nil
}

func (m *Member) Request_subscription(plan_id string) error {
	if m.Customer() == nil || m.customer.DefaultSource == nil {
		return fmt.Errorf("No valid payment source")
	}
	if _, err := m.Exec(
		"INSERT INTO pending_subscription "+
		"(member, plan_id) "+
		"VALUES ($1, $2) "+
		"ON CONFLICT (member, plan_id) DO UPDATE "+
		"SET plan_id = $2", m.Id, plan_id); err != nil {
		log.Panic(err)
	}
	return nil
}

func (ms *Members) Approved_by(sub *stripe.Sub) *Member {
	approved_by, err := strconv.Atoi(sub.Meta["approved_by"])
	if err != nil {
		log.Panic(err)
	}
	return ms.Get_member_by_id(approved_by)
}

func (ms *Members) Created_at(sub *stripe.Sub) time.Time {
	return time.Unix(sub.Created, 0)
}

func (ms *Members) load_plans() {
	i := plan.List(nil)
	for i.Next() {
		p := i.Plan()
		ms.Plans[p.ID] = p
	}
}
