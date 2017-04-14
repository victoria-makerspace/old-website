package member

import (
	"fmt"
	"strings"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/sub"
	"log"
)

func (m *Member) Request_membership() error {
	plan := m.Plans["membership-regular"]
	if m.Student != nil {
		plan = m.Plans["membership-student"]
	}
	return m.Request_subscription(plan.ID)
}

func (m *Member) Get_pending_membership() *Pending_subscription {
	for _, p := range m.Get_pending_subscriptions() {
		if strings.HasPrefix(p.Plan_id, "membership") {
			return p
		}
	}
	return nil
}

func (m *Member) Get_membership() *stripe.Sub {
	if m.Customer() == nil {
		return nil
	}
	for _, s := range m.customer.Subs.Values {
		if strings.HasPrefix(s.Plan.ID, "membership") {
			return s
		}
	}
	return nil
}

func (m *Member) Update_membership(params *stripe.SubParams) error {
	ms := m.Get_membership()
	if ms == nil {
		return fmt.Errorf("No existing membership")
	}
	_, err := sub.Update(ms.ID, params)
	return err
}

func (m *Member) Cancel_membership() {
	ms := m.Get_membership()
	if m != nil {
		_, err := sub.Cancel(ms.ID, nil)
		if err != nil {
			log.Println(err)
		}
	}
	if m.Talk_user() != nil {
		m.Talk_user().Remove_from_group("Members")
	}
}
