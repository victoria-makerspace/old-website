package member

import (
	"database/sql"
	"fmt"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/sub"
	"log"
	"sort"
	"strings"
)

type Storage struct {
	Number    int
	Quantity  uint64
	Available bool
	*stripe.Plan
	sub_id string
	subitem_id string
	*Member
}

func (ms *Members) get_storage(plan_id string, number int) (*Storage, error) {
	p, ok := ms.Plans[plan_id]
	if !ok || Plan_category(p.ID) != "storage" {
		return nil, fmt.Errorf("Invalid storage plan '%s'", plan_id)
	}
	s := &Storage{Number: number, Plan: p}
	var sub_id, subitem_id sql.NullString
	if err := ms.QueryRow(
		"SELECT"+
		"	available,"+
		"	quantity,"+
		"	subscription_id,"+
		"	subitem_id "+
		"FROM storage "+
		"WHERE plan_id = $1 AND number = $2",
		plan_id, number).Scan(&s.Available, &s.Quantity,
			&sub_id, &subitem_id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("Invalid storage number for '%s'", p.Name)
		}
		log.Panic(err)
	}
	if sub_id.Valid {
		sb, err := sub.Get(sub_id.String, nil)
		if err != nil {
			log.Panic(err)
		}
		s.sub_id = sub_id.String
		s.subitem_id = subitem_id.String
		s.Member = ms.Get_member_by_customer_id(sb.Customer.ID)
	}
	return s, nil
}

func (ms *Members) List_storage_plans() []string {
	plans := make([]string, 0)
	for plan_id, p := range ms.Plans {
		if Plan_category(p.ID) == "storage" {
			plans = append(plans, plan_id)
		}
	}
	sort.Strings(plans)
	return plans
}

func (ms *Members) List_storage(plan_id string) ([]*Storage, error) {
	if p, ok := ms.Plans[plan_id]; !ok {
		return nil, fmt.Errorf("Invalid plan ID")
	} else if Plan_category(p.ID) != "storage" {
		return nil, fmt.Errorf("Invalid storage plan ID")
	}
	storage := make([]*Storage, 0)
	rows, err := ms.Query(
		"SELECT"+
		"	number, quantity, available, subscription_id, subitem_id  "+
		"FROM storage "+
		"WHERE plan_id = $1 "+
		"ORDER BY number ASC", plan_id)
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var s Storage
		var sub_id, subitem_id sql.NullString
		if err = rows.Scan(&s.Number, &s.Quantity, &s.Available, &sub_id,
			&subitem_id); err != nil {
			log.Panic(err)
		}
		s.Plan = ms.Plans[plan_id]
		if sub_id.Valid {
			sb, err := sub.Get(sub_id.String, nil)
			if err != nil {
				log.Panic(err)
			}
			s.sub_id = sub_id.String
			s.subitem_id = subitem_id.String
			s.Member = ms.Get_member_by_customer_id(sb.Customer.ID)
		}
		storage = append(storage, &s)
	}
	return storage, nil
}

func (ms *Members) List_pending_storage_leases() []*Pending_subscription {
	pending := make([]*Pending_subscription, 0)
	rows, err := ms.Query(
		"SELECT member, requested_at, plan_id " +
			"FROM pending_subscription " +
			"ORDER BY requested_at DESC")
	defer rows.Close()
	if err != nil && err != sql.ErrNoRows {
		log.Panic(err)
	}
	for rows.Next() {
		var p Pending_subscription
		var member_id int
		if err = rows.Scan(&member_id, &p.Requested_at, &p.Plan_id); err != nil {
			log.Panic(err)
		}
		if !strings.HasPrefix((p.Plan_id), "storage-") {
			continue
		}
		p.Member = ms.Get_member_by_id(member_id)
		pending = append(pending, &p)
	}
	return pending
}

func (m *Member) List_storage_leases_by_plan(plan_id string) ([]*Storage, error) {
	storage := make([]*Storage, 0)
	st_numbers, err := m.List_storage(plan_id)
	if err != nil {
		return nil, err
	}
	for _, st := range st_numbers {
		if st.Member != nil && st.Member.Id == m.Id {
			st.Member = m
			storage = append(storage, st)
		}
	}
	return storage, nil
}

func (m *Member) New_storage_lease(plan_id string, number int) error {
	st_numbers, err := m.List_storage_leases_by_plan(plan_id)
	if err != nil {
		return err
	}
	quantity := uint64(0)
	var sub_id, subitem_id string
	for _, st := range st_numbers {
		sub_id = st.sub_id
		subitem_id = st.subitem_id
		quantity += st.Quantity
	}
	lease, err := m.get_storage(plan_id, number)
	if err != nil {
		return err
	}
	if lease.Member != nil {
		return fmt.Errorf("%s number %d already has an active lease by member "+
			"@%s", lease.Plan.Name, number, lease.Member.Username)
	}
	quantity += lease.Quantity
	if len(st_numbers) > 0 {
		if err := m.Update_subscription_item(sub_id, subitem_id, quantity);
			err != nil {
			return err
		}
	} else {
		item, sb, err := m.New_subscription_item(plan_id, lease.Quantity)
		if err != nil {
			return err
		}
		sub_id = sb.ID
		subitem_id = item.ID
	}
	if _, err = m.Exec(
		"UPDATE storage "+
			"SET subscription_id = $3, subitem_id = $4 "+
			"WHERE plan_id = $1 AND number = $2",
			plan_id, number, sub_id, subitem_id); err != nil {
		log.Panic(err)
	}
	return nil
}

func (m *Member) Cancel_storage_lease(plan_id string, number int) error {
	st_numbers, err := m.List_storage_leases_by_plan(plan_id)
	if err != nil {
		return err
	}
	var lease *Storage
	quantity := uint64(0)
	for _, st := range st_numbers {
		if st.Number == number {
			lease = st
			continue
		}
		quantity += st.Quantity
	}
	if lease == nil {
		return fmt.Errorf("%s number %d is not currently leased by @%s",
			m.Plans[plan_id].Name, number, m.Id)
	}
	if _, err = m.Exec(
		"UPDATE storage "+
			"SET subscription_id = NULL, subitem_id = NULL "+
			"WHERE plan_id = $1 AND number = $2", plan_id, number); err != nil {
		log.Panic(err)
	}
	//TODO: Update_storage_waitlist
	return m.Update_subscription_item(lease.sub_id, lease.subitem_id, quantity)
}
