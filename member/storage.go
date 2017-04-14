package member

import (
	"database/sql"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/sub"
	"log"
	"strconv"
)

type Storage struct {
	Number    int
	Size      int
	Available bool
	*stripe.Plan
	*Member
}

func (ms *Members) Get_storage(plan_id string) []*Storage {
	storage := make([]*Storage, 0)
	i := sub.List(&stripe.SubListParams{Plan: plan_id})
	members := make(map[int]*Member)
	for i.Next() {
		number, err := strconv.Atoi(i.Sub().Meta["number"])
		if err != nil {
			log.Panic(err)
		}
		members[number] =  ms.Get_member_by_customer_id(i.Sub().Customer.ID)
	}
	rows, err := ms.Query("SELECT number, size, available "+
		"FROM storage WHERE plan_id = $1 ORDER BY number ASC", plan_id)
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			number     int
			size       sql.NullInt64
			available  bool
		)
		if err = rows.Scan(&number, &size, &available); err != nil {
			log.Panic(err)
		}
		s := &Storage{
			Number:    number,
			Size:      int(size.Int64),
			Available: available,
			Plan:      ms.Plans[plan_id]}
		if m, ok := members[number]; ok {
			s.Member = m
		}
		storage = append(storage, s)
	}
	return storage
}
