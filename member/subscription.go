package member

import (
	"database/sql"
	"fmt"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/sub"
	"github.com/stripe/stripe-go/subitem"
	"log"
	"time"
)

type Pending_subscription struct {
	*Member
	Requested_at time.Time
	Plan_id      string
}

func (m *Member) Request_subscription(plan string) error {
	p, ok := m.Plans[plan]
	if !ok {
		return fmt.Errorf("Invalid plan identifier: " + plan)
	}
	if !m.Has_card() && p.Amount != 0 {
		return fmt.Errorf("No valid payment source")
	}
	if _, err := m.Exec(
		"INSERT INTO pending_subscription "+
			"(member, plan_id) "+
			"VALUES ($1, $2) "+
			"ON CONFLICT (member, plan_id) DO NOTHING", m.Id, plan); err != nil {
		log.Panic(err)
	}
	return nil
}

func (m *Member) Get_pending_subscriptions() []*Pending_subscription {
	pending := make([]*Pending_subscription, 0)
	rows, err := m.Query(
		"SELECT requested_at, plan_id "+
			"FROM pending_subscription "+
			"WHERE member = $1 "+
			"ORDER BY requested_at DESC", m.Id)
	defer rows.Close()
	if err != nil && err != sql.ErrNoRows {
		log.Panic(err)
	}
	for rows.Next() {
		p := Pending_subscription{Member: m}
		if err = rows.Scan(&p.Requested_at, &p.Plan_id); err != nil {
			log.Panic(err)
		}
		pending = append(pending, &p)
	}
	return pending
}

func (ms *Members) Cancel_pending_subscription(p *Pending_subscription) {
	if _, err := ms.Exec(
		"DELETE FROM pending_subscription "+
			"WHERE member = $1 AND plan_id = $2", p.Member.Id, p.Plan_id); err != nil {
		log.Panic(err)
	}
}

func (m *Member) get_subscriptions() {
	subs := make(map[string]*stripe.Sub)
	for _, s := range m.customer.Customer.Subs.Values {
		if s.Ended == 0 {
			subs[s.ID] = s
		}
	}
	m.customer.Subscriptions = subs
}

func (m *Member) get_subscription_by_interval(interval string) *stripe.Sub {
	for _, s := range m.customer.Subscriptions {
		if Subscription_interval(s) == interval {
			return s
		}
	}
	return nil
}

func Subscription_interval(s *stripe.Sub) string {
	if s.Plan != nil {
		return Plan_interval(s.Plan)
	}
	return Plan_interval(s.Items.Values[0].Plan)
}

func (m *Member) cancel_subscription(id string) error {
	if _, ok := m.customer.Subscriptions[id]; !ok {
		return fmt.Errorf("Non-existant subscription ID for @%s", m.Username)
	}
	_, err := sub.Cancel(id, nil)
	return err
}

func (m *Member) Get_subscription_from_item(item_id string) (*stripe.Sub, error) {
	for _, s := range m.customer.Subscriptions {
		for _, item := range s.Items.Values {
			if item.ID == item_id {
				return s, nil
			}
		}
	}
	return nil, fmt.Errorf("Non-existant item ID for @%s", m.Username)
}

func (m *Member) New_subscription_item(plan_id string, quantity int) error {
	p, ok := m.Plans[plan_id]
	if !ok {
		return fmt.Errorf("Invalid plan '%s'", plan_id)
	}
	if !m.Has_card() {
		if p.Amount != 0 {
			return fmt.Errorf("No valid payment source")
		} else if err := m.Update_customer("", nil); err != nil {
			return err
		}
	}
	s := m.get_subscription_by_interval(Plan_interval(p))
	if s == nil {
		sub_params := &stripe.SubParams{
			Customer: m.Customer_id,
			Items: []*stripe.SubItemsParams{&stripe.SubItemsParams{
				Plan:     p.ID,
				Quantity: uint64(quantity)}}}
		_, err := sub.New(sub_params)
		return err
	}
	item_params := &stripe.SubItemParams{
		Sub:      s.ID,
		Plan:     p.ID,
		Quantity: uint64(quantity)}
	_, err := subitem.New(item_params)
	return err
}

func (m *Member) Cancel_subscription_item(sub_id, item_id string) error {
	s, ok := m.customer.Subscriptions[sub_id]
	if !ok {
		return fmt.Errorf("Invalid subscription ID")
	}
	for _, i := range s.Items.Values {
		if i.ID != item_id {
			continue
		}
		if len(s.Items.Values) == 1 && s.Plan == nil {
			return m.cancel_subscription(sub_id)
		}
		_, err := subitem.Del(item_id, nil)
		return err
	}
	return fmt.Errorf("Non-existant subscription ID for @%s", m.Username)
}
