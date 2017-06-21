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

//TODO: check for duplicate requests
func (m *Member) Request_subscription(plan string) error {
	p, ok := m.Plans[plan]
	if !ok {
		return fmt.Errorf("Invalid plan identifier: " + plan)
	}
	if m.Get_payment_source() == nil && p.Amount != 0 {
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

func (m *Member) Get_pending_subscription_by_plan(plan_id string) *Pending_subscription {
	pending := &Pending_subscription{Member: m, Plan_id: plan_id}
	if err := m.QueryRow(
		"SELECT requested_at "+
		"FROM pending_subscription "+
		"WHERE member = $1 AND plan_id = $2", m.Id, plan_id).Scan(
		&pending.Requested_at); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
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

func (ms *Members) Number_pending(plan string) int {
	var num int
	if err := ms.QueryRow(
		"SELECT COUNT(*) FROM pending_subscription WHERE plan_id = $1",
		ms.Plans[plan]).Scan(&num); err != nil {
		log.Panic(err)
	}
	return num
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
	for _, s := range m.Get_customer().Subscriptions {
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
	if _, ok := m.Get_customer().Subscriptions[id]; !ok {
		return fmt.Errorf("Non-existant subscription ID for @%s", m.Username)
	}
	_, err := sub.Cancel(id, nil)
	return err
}

func (m *Member) Get_subscription_from_item(item_id string) (*stripe.Sub, error) {
	for _, s := range m.Get_customer().Subscriptions {
		for _, item := range s.Items.Values {
			if item.ID == item_id {
				return s, nil
			}
		}
	}
	return nil, fmt.Errorf("Non-existant item ID for @%s", m.Username)
}

func (m *Member) New_subscription_item(plan_id string, quantity uint64) (*stripe.SubItem, *stripe.Sub, error) {
	p, ok := m.Plans[plan_id]
	if !ok {
		return nil, nil, fmt.Errorf("Invalid plan '%s'", plan_id)
	}
	if m.Get_payment_source() == nil {
		if p.Amount != 0 {
			return nil, nil, fmt.Errorf("No valid payment source")
		} else if err := m.Update_customer(""); err != nil {
			return nil, nil, err
		}
	}
	s := m.get_subscription_by_interval(Plan_interval(p))
	if s == nil {
		sub_params := &stripe.SubParams{
			Customer: m.Customer_id,
			Items: []*stripe.SubItemsParams{&stripe.SubItemsParams{
				Plan:     p.ID,
				Quantity: quantity}}}
		s, err := sub.New(sub_params)
		if err != nil {
			return nil, nil, err
		}
		m.Get_customer().Subscriptions[s.ID] = s
		return s.Items.Values[0], s, nil
	}
	//TODO: update m.Get_customer().Subscriptions with new subitem
	item_params := &stripe.SubItemParams{
		Sub:      s.ID,
		Plan:     p.ID,
		Quantity: quantity}
	subitem, err := subitem.New(item_params)
	if err != nil {
		return nil, nil, err
	}
	return subitem, s, nil
}

func (m *Member) Update_subscription_item(sub_id, subitem_id string, quantity uint64) error {
	if quantity == 0 {
		return m.Cancel_subscription_item(sub_id, subitem_id)
	}
	sub, ok := m.Get_customer().Subscriptions[sub_id]
	if !ok {
		return fmt.Errorf("Invalid subscription ID for @%s", m.Username)
	}
	params := &stripe.SubItemParams{Quantity: quantity}
	subitem, err := subitem.Update(subitem_id, params)
	if err != nil {
		return err
	}
	for i, si := range sub.Items.Values {
		if si.ID != subitem_id {
			continue
		}
		sub.Items.Values[i] = subitem
		return nil
	}
	return fmt.Errorf("Invalid subscription item ID for @%s", m.Username)
}

func (m *Member) Cancel_subscription_item(sub_id, item_id string) error {
	s, ok := m.Get_customer().Subscriptions[sub_id]
	if !ok {
		return fmt.Errorf("Invalid subscription ID for @%s", m.Username)
	}
	for i, item := range s.Items.Values {
		if item.ID != item_id {
			continue
		}
		if len(s.Items.Values) == 1 {
			if err := m.cancel_subscription(sub_id); err != nil {
				return err
			}
			delete(m.customer.Subscriptions, sub_id)
			return nil
		}
		if _, err := subitem.Del(item_id, nil); err != nil {
			return err
		}
		s.Items.Values = append(s.Items.Values[:i], s.Items.Values[i+1:]...)
		return nil
	}
	return fmt.Errorf("Invalid subscription item ID for @%s", m.Username)
}
