package member

import (
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/sub"
	"log"
	"fmt"
	"strconv"
	"time"
)

type Pending_subscription struct {
	*Member
	Requested_at time.Time
	Plan_id string
}

func (m *Member) Request_subscription(plan string) error {
	if m.Customer() == nil || m.customer.DefaultSource == nil {
		return fmt.Errorf("No valid payment source")
	}
	p, ok := m.Plans[plan]
	if !ok {
		return fmt.Errorf("Invalid plan identifier: " + plan)
	}
	if _, err := m.Exec(
		"INSERT INTO pending_subscription "+
		"(member, plan_id) "+
		"VALUES ($1, $2) "+
		"ON CONFLICT (member, plan_id) DO NOTHING", m.Id, p.ID);
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
