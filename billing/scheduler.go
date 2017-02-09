package billing

import (
	"database/sql"
	"log"
	"strconv"
	"time"
)

// first_of_next_month returns the time at 00:00 on the first day of next month
func first_of_next_month() time.Time {
	return time.Date(time.Now().Year(), time.Now().Month()+1, 1, 0, 0, 0, 0, time.Local)
}

func (b *Billing) payment_scheduler() {
	for {
		t := monthly_timer()
		<-t.C
		go func() {
			log.Println("Starting payment scheduler")
			defer log.Println("Payment scheduler completed")
			// Left join in case anyone ever ends up with a recurring billing without a beanstream profile
			// TODO: implement the billing.monthly flag to actually do something
			rows, err := b.db.Query("SELECT b.username, b.name, b.amount, bp.id FROM billing b LEFT JOIN billing_profile bp USING username WHERE b.monthly = true AND (b.end_date > now() OR b.end_date IS NULL)")
			defer rows.Close()
			if err != nil {
				if err != sql.ErrNoRows {
					log.Panic(err)
				}
				return
			}
			type payment struct {
				username string
				name     string
				amount   float64
			}
			var (
				payments   []payment
				members    map[string]*Profile
				profile_id string
				a          string
			)
			for rows.Next() {
				pmnt := payment{}
				if err = rows.Scan(&pmnt.username, &pmnt.name, &a, &profile_id); err != nil {
					log.Panic(err)
				}
				if pmnt.amount, err = strconv.ParseFloat(a[1:], 32); err != nil {
					log.Println(err)
				}
				payments = append(payments, pmnt)
				// Ensure no redundant profile queries are sent to Beanstream
				if _, ok := members[pmnt.username]; profile_id != "" && !ok {
					if prof := b.Get_profile(profile_id); prof != nil {
						members[pmnt.username] = prof
					}
				}
				// Start transactions
			}
		}()
	}
}

// Fires an event on the first of every month, first thing in the morning
//	(00:00) at local time
func monthly_timer() *time.Timer {
	return time.NewTimer(first_of_next_month().Sub(time.Now()))
}
