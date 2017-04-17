package member

import (
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"
	"github.com/stripe/stripe-go/plan"
	"github.com/stripe/stripe-go/sub"
	"log"
	"fmt"
	"strconv"
	"time"
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

type Pending_subscription struct {
	*Member
	Requested_at time.Time
	Plan_id string
}

func (m *Member) Request_subscription(plan_id string) error {
	if m.Customer() == nil || m.customer.DefaultSource == nil {
		return fmt.Errorf("No valid payment source")
	}
	if _, err := m.Exec(
		"INSERT INTO pending_subscription "+
		"(member, plan_id) "+
		"VALUES ($1, $2) "+
		"ON CONFLICT (member, plan_id) DO NOTHING", m.Id, plan_id);
		err != nil {
		log.Panic(err)
	}
	return nil
}

func (ms *Members) Cancel_pending_subscription(p *Pending_subscription) {
	if _, err := ms.Exec(
		"DELETE FROM pending_subscription "+
		"WHERE member = $1 AND plan_id = $2", p.Member.Id, p.Plan_id);
		err != nil {
		log.Panic(err)
	}
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

func (m *Member) Cancel_subscription(id string) error {
	if _, ok := m.Active_subscriptions()[id]; !ok {
		return fmt.Errorf("Invalid subscription ID")
	}
	_, err := sub.Cancel(id, nil)
	return err
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
