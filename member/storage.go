package member

import (
	"database/sql"
	"fmt"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/sub"
	"log"
	"sort"
)

type Storage struct {
	Number    int
	Quantity  int
	Available bool
	*stripe.Plan
	sub_id string
	subitem_id string
	*Member
}

func (ms *Members) get_storage_number(plan_id string, number int) (*Storage, error) {
	p, ok := ms.Plans[plan_id]
	if !ok || Plan_category(p.ID) != "storage" {
		return nil, fmt.Errorf("Invalid storage plan '%s'", plan_id)
	}
	s := &Storage{Number: number, Plan: p}
	var sub_id, subitem_id sql.NullString
	if err := ms.QueryRow(
		"SELECT quantity, available, subscription_id, subitem_id "+
			"FROM storage "+
			"WHERE plan_id = $1 AND number = $2",
		plan_id, number).Scan(s.Quantity, s.Available, &sub_id, &subitem_id);
		err != nil {
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

func (ms *Members) List_storage(plan_id string) []*Storage {
	storage := make([]*Storage, 0)
	rows, err := ms.Query(
		"SELECT number, quantity, available, subscription_id "+
			"FROM storage WHERE plan_id = $1 ORDER BY number ASC", plan_id)
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var s Storage
		var sub_id, subitem_id sql.NullString
		if err = rows.Scan(&s.Number, &s.Quantity, &s.Available, &sub_id); err != nil {
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
	return storage
}

func (m *Member) New_storage_lease(plan_id string, number int) error {
	s, err := m.get_storage_number(plan_id, number)
	if err != nil {
		return err
	}
	if s.Member != nil {
		return fmt.Errorf("%s number %d already has an active lease by member "+
			"@%s", s.Plan.Name, number, s.Member.Username)
	}
	item, sb, err := m.New_subscription_item(plan_id, s.Quantity)
	if err != nil {
		return err
	}
	if _, err = m.Exec(
		"UPDATE storage "+
			"SET subscription_id = $3, subitem_id = $4 "+
			"WHERE plan_id = $1 AND number = $2",
			plan_id, number, sb.ID, item.ID); err != nil {
		log.Panic(err)
	}
	return nil
}

func (m *Member) Cancel_storage_lease(plan_id string, number int) error {
	s, err := m.get_storage_number(plan_id, number)
	if err != nil {
		return err
	}
	if s.Member == nil || s.Member.Id != m.Id {
		return fmt.Errorf("%s number %d is not currently leased by @%s",
			s.Plan.Name, number, m.Id)
	}
	if err = m.Cancel_subscription_item(s.sub_id, s.subitem_id); err != nil {
		return err
	}
	if _, err = m.Exec(
		"UPDATE storage "+
			"SET subscription_id = NULL, subitem_id = NULL "+
			"WHERE plan_id = $1 AND number = $2", plan_id, number); err != nil {
		log.Panic(err)
	}
	return nil
}
