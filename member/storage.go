package member

import (
	"database/sql"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/sub"
	"log"
	"sort"
)

type Storage struct {
	Number    int
	Quantity  uint64
	Available bool
	*stripe.Plan
	*Member
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
		var sub_id sql.NullString
		if err = rows.Scan(&s.Number, &s.Quantity, &s.Available, &sub_id);
			err != nil {
			log.Panic(err)
		}
		s.Plan = ms.Plans[plan_id]
		if sub_id.Valid {
			sb, err := sub.Get(sub_id.String, nil)
			if err != nil {
				log.Panic(err)
			}
			s.Member = ms.Get_member_by_customer_id(sb.Customer.ID)
		}
		storage = append(storage, &s)
	}
	return storage
}



func (m *Member) New_storage_lease(plan_id string, number int) error {

	return nil
}
