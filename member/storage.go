package member

import (
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/sub"
	"log"
	"sort"
	"strconv"
	"strings"
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
	for p, _ := range ms.Plans {
		if strings.HasPrefix(p, "storage") {
			plans = append(plans, p)
		}
	}
	sort.Strings(plans)
	return plans
}

func (ms *Members) List_storage(plan_id string) []*Storage {
	storage := make([]*Storage, 0)
	i := sub.List(&stripe.SubListParams{Plan: plan_id})
	members := make(map[int]*Member)
	for i.Next() {
		n, ok := i.Sub().Meta["number"]
		if !ok {
			continue
		}
		number, err := strconv.Atoi(n)
		if err != nil {
			log.Panic(err)
		}
		m := ms.Get_member_by_customer_id(i.Sub().Customer.ID)
		if m != nil {
			members[number] = m
		} else {
			log.Printf("Unregistered customer <%s> subscribed to %s number %d",
				i.Sub().Customer.ID, plan_id, number)
		}
	}
	rows, err := ms.Query("SELECT number, quantity, available "+
		"FROM storage WHERE plan_id = $1 ORDER BY number ASC", plan_id)
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var s Storage
		if err = rows.Scan(&s.Number, &s.Quantity, &s.Available); err != nil {
			log.Panic(err)
		}
		s.Plan = ms.Plans[plan_id]
		if m, ok := members[s.Number]; ok {
			s.Member = m
		}
		storage = append(storage, &s)
	}
	return storage
}
