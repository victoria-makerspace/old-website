package member

import (
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/sub"
	"log"
	"strconv"
)

type Storage struct {
	Number    int
	Quantity  uint64
	Available bool
	*stripe.Plan
	*Member
}

func (ms *Members) Get_storage(plan_id string) []*Storage {
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
		members[number] =  ms.Get_member_by_customer_id(i.Sub().Customer.ID)
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
