package member

import (
	"fmt"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/sub"
	"github.com/stripe/stripe-go/subitem"
	"log"
	"strings"
)

func (m *Member) Request_membership(rate string) error {
	p, ok := m.Plans["membership-"+rate]
	if !ok {
		return fmt.Errorf("Invalid membership rate")
	}
	if rate == "student" && m.Student == nil {
		return fmt.Errorf("Non-students cannot apply for a student membership" +
			" rate")
	}
	if mp := m.Get_membership(); mp != nil {
		if mp.Plan.ID == p.ID {
			return fmt.Errorf("Cannot duplicate existing membership")
		}
		if mp.Plan.Amount < p.Amount {
			return m.Update_membership(rate)
		}
	}
	return m.Request_subscription("membership-" + rate)
}

func (m *Member) Get_pending_membership() *Pending_subscription {
	for _, ps := range m.Get_pending_subscriptions() {
		if strings.HasPrefix(ps.Plan_id, "membership-") {
			return ps
		}
	}
	return nil
}

func (m *Member) Get_membership() *stripe.SubItem {
	if m.Get_customer() == nil {
		return nil
	}
	for _, s := range m.customer.Subscriptions {
		for _, i := range s.Items.Values {
			if Plan_category(i.Plan.ID) == "membership" {
				return i
			}
		}
	}
	return nil
}

func (m *Member) Membership_rate() string {
	if ms := m.Get_membership(); ms != nil {
		return Plan_identifier(ms.Plan.ID)
	}
	return ""
}

func (m *Member) Membership_id() string {
	if ms := m.Get_membership(); ms != nil {
		return ms.ID
	}
	return ""
}

func (m *Member) Update_membership(rate string) error {
	p, ok := m.Plans["membership-"+rate]
	if !ok {
		return fmt.Errorf("Invalid membership rate '%s'", rate)
	}
	mp := m.Get_membership()
	if mp == nil {
		return fmt.Errorf("No existing membership")
	}
	_, err := subitem.Update(mp.ID, &stripe.SubItemParams{Plan: p.ID})
	return err
}

func (m *Member) Cancel_membership() {
	mp := m.Get_membership()
	if mp != nil {
		s, err := m.Get_subscription_from_item(mp.ID)
		if err != nil {
			log.Println(err)
		} else if err = m.Cancel_subscription_item(s.ID, mp.ID); err != nil {
			log.Println(err)
		}
	}
	if m.Talk_user() != nil {
		m.Talk_user().Remove_from_group("Members")
	}
}

// Indexed by customer ID
func (ms *Members) List_all_memberships() map[string]*stripe.Sub {
	subs := make(map[string]*stripe.Sub)
	for _, p := range ms.Plans {
		if Plan_category(p.ID) != "membership" {
			continue
		}
		i := sub.List(&stripe.SubListParams{Plan: p.ID})
		for i.Next() {
			s := i.Sub()
			subs[s.Customer.ID] = s
		}
	}
	return subs
}

// Indexed by customer ID
func (ms *Members) List_memberships(plan_id string) map[string]*stripe.Sub {
	subs := make(map[string]*stripe.Sub)
	p, ok := ms.Plans[plan_id]
	if !ok || Plan_category(p.ID) != "membership" {
		log.Panic("Invalid membership plan")
	}
	i := sub.List(&stripe.SubListParams{Plan: p.ID})
	for i.Next() {
		s := i.Sub()
		subs[s.Customer.ID] = s
	}
	return subs
}
