package billing

import (
	"database/sql"
	"log"
	"strconv"
	"time"
	"github.com/vvanpo/makerspace/member"
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
			//	transaction should occur.  Left join billing_profile in case
			//	anyone ever ends up with a recurring billing without a
			//	beanstream profile.
			// TODO: implement the billing.monthly flag to actually do something
			rows, err := b.db.Query("SELECT b.username, b.name, b.amount, bp.id FROM billing b LEFT JOIN billing_profile bp USING (username) WHERE b.monthly = true AND (b.end_date > now() OR b.end_date IS NULL)")
			defer rows.Close()
			if err != nil {
				if err != sql.ErrNoRows {
					log.Panic(err)
				}
				return
			}
			type payment struct {
				member	 *member.Member
				name     string
				amount   float64
			}
			var (
				payments   []payment
				members    map[string]*member.Member
				profiles   map[string]*Profile
				username   string
				profile_id string
				a          string	// intermediate 'amount' string before
									//	conversion to float
			)
			for rows.Next() {
				pmnt := payment{}
				if err = rows.Scan(&username, &pmnt.name, &a, &profile_id); err != nil {
					log.Panic(err)
				}
				if profile_id == "" {
					//// TODO: missed payment
				}
				// Convert amount to float
				if pmnt.amount, err = strconv.ParseFloat(a[1:], 32); err != nil {
					log.Println(err)
				}
				payments = append(payments, pmnt)
				// Ensure we only query the database once per member
				if _, ok := members[username]; !ok {
					members[username] = member.Get(username, b.db)
				}
				// Ensure no redundant profile queries are sent to Beanstream
				if _, ok := profiles[username]; !ok {
					if profile := b.Get_profile(members[username]); profile != nil {
						profiles[username] = profile
					} else {
						log.Println("Could not fetch Beanstream profile for " + username)
						//// TODO: missed payment
					}
				}
				// Do transaction
				
			}
		}()
	}
}

// Fires an event on the first of every month, first thing in the morning
//	(00:00) at local time
func monthly_timer() *time.Timer {
	return time.NewTimer(first_of_next_month().Sub(time.Now()))
}
