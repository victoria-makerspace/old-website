package billing

import (
	"database/sql"
	"github.com/vvanpo/makerspace/member"
	"log"
	"time"
)

// first_of_next_month returns the local time at 00:00 on the first day of next
//	month
func first_of_next_month() time.Time {
	return time.Date(time.Now().Year(), time.Now().Month()+1, 1, 0, 0, 0, 0, time.Local)
}

func (b *Billing) payment_scheduler() {
	for {
		t := monthly_timer()
		<-t.C
		go func() {
			// TODO: ensure the scheduler hasn't already run for this month
			log.Println("Starting payment scheduler")
			defer log.Println("Payment scheduler completed")
			// Query to find all open billing registrations for which a
			//	transaction should occur.
			//TODO: recurring intervals different than 1 month
			rows, err := b.db.Query("SELECT i.id, i.profile, COALESCE(i.amount, f.amount) FROM invoice i INNER JOIN fee f ON (i.fee = f.id) WHERE f.recurring = '1 month' AND (i.end_date >= now() OR i.end_date IS NULL)")
			if err != nil {
				if err != sql.ErrNoRows {
					log.Panic(err)
				}
				return
			}
			defer rows.Close()
			type payment struct {
				id int
				profile *Profile
				amount float64
			}
			var (
				payments   []payment
				members    map[string]*member.Member
				profiles   map[string]*Profile
				profile_username string
			)
			for rows.Next() {
				pmnt := payment{}
				if err = rows.Scan(&pmnt.id, &profile_username, &pmnt.amount); err != nil {
					log.Panic(err)
				}
				if _, ok := members[profile_username]; !ok {
					members[profile_username] = member.Get(profile_username, b.db)
				}
				// Ensure no redundant profile queries are sent to Beanstream
				if _, ok := profiles[profile_username]; !ok {
					if profile := b.Get_profile(members[profile_username]); profile != nil {
						profiles[profile_username] = profile
					} else {
						log.Println("Could not fetch Beanstream profile for " + username)
						//// TODO: missed payment
					}
				}
				// Do transaction

				payments = append(payments, pmnt)
			}
		}()
	}
}

// Fires an event on the first of every month, first thing in the morning
//	(00:00) at local time
func monthly_timer() *time.Timer {
	return time.NewTimer(first_of_next_month().Sub(time.Now()))
}
