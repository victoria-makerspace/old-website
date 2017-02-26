package billing

import (
	"database/sql"
	"log"
	"time"
)

//TODO: recurring intervals different than 1 month
func (b *Billing) payment_scheduler() {
	for {
		run := func(interval string) {
			var sched_error string
			txn_todo := b.count_recurring(interval)
			txn_attempts := 0
			txn_approved := 0
			txn_log := b.log_scheduled(interval, txn_todo)
			log.Printf("(%d) Starting payment scheduler (%s): %d txns\n",
				txn_log, interval, txn_todo)
			defer func() {
				b.log_error(txn_log, txn_attempts, txn_approved, sched_error)
				err := ""
				if sched_error != "" {
					err = "\n\tError: " + sched_error
				}
				log.Printf("(%d) Payment scheduler (%s) completed:\n"+
					"\t%d scheduled\n"+
					"\t%d attempted\n"+
					"\t%d approved%s\n",
					txn_log, interval, txn_todo, txn_attempts, txn_approved,
					err)
			}()
			profiles := b.get_all_profiles()
			for _, p := range profiles {
				for _, inv := range p.Invoices {
					txn_attempts += 1
					txn := p.do_recurring_txn(inv)
					txn.log_recurring_txn(txn_log)
					if txn.Approved {
						txn_approved += 1
					}
				}
			}
			sched_error = ""
		}
		//TODO: for intervals := b.get_intervals() {
		if !b.has_run("1 month") {
			go run("1 month")
		}
		t := monthly_timer()
		<-t.C
		go run("1 month")
	}
}

func (b *Billing) log_scheduled(interval string, txn_todo int) int {
	var log_id int
	if err := b.db.QueryRow("INSERT INTO txn_scheduler_log (interval, txn_todo) "+
		"VALUES ($1, $2) RETURNING id", interval, txn_todo).Scan(&log_id); err != nil {
		log.Panic(err)
	}
	return log_id
}

func (b *Billing) log_error(log_id, txn_attempts, txn_approved int, e string) {
	if e == "" {
		return
	}
	if _, err := b.db.Exec(
		"UPDATE txn_scheduler_log "+
			"SET "+
			"	txn_attempts = $2, "+
			"	txn_approved = $3, "+
			"	error = $4, "+
			"WHERE id = $1",
		log_id, txn_attempts, txn_approved, e); err != nil {
		log.Panic(err)
	}
}

func (b *Billing) get_intervals() []string {
	ints := make([]string, 0)
	rows, err := b.db.Query(
		"SELECT COALESCE(i.recurring, f.recurring) rc" +
			"FROM invoice i " +
			"LEFT JOIN fee f " +
			"ON i.fee = f.id " +
			"WHERE COALESCE(i.recurring, f.recurring) IS NOT NULL " +
			"	AND (i.end_date > now() OR i.end_date IS NULL) " +
			"GROUP BY rc")
	defer rows.Close()
	if err != nil {
		if err == sql.ErrNoRows {
			return ints
		}
		log.Panic(err)
	}
	for rows.Next() {
		var i string
		rows.Scan(&i)
		ints = append(ints, i)
	}
	return ints
}

func (b *Billing) count_recurring(interval string) int {
	var count int
	if err := b.db.QueryRow(
		"SELECT COUNT(*) "+
			"FROM invoice i "+
			"LEFT JOIN fee f "+
			"ON (i.fee = f.id) "+
			"WHERE "+
			"	COALESCE(i.recurring, f.recurring) = $1 "+
			"	AND (i.end_date > now() OR i.end_date IS NULL)",
		interval).Scan(&count); err != nil {
		log.Panic(err)
	}
	return count
}

func (b *Billing) has_run(interval string) bool {
	var has_run bool
	if err := b.db.QueryRow(
		"SELECT NOT age(time) > $1 "+
			"FROM txn_scheduler_log "+
			"WHERE interval = $1 "+
			"ORDER BY time DESC "+
			"LIMIT 1", interval).Scan(&has_run); err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		log.Panic(err)
	}
	return has_run
}

// first_of_next_month returns the local time at 00:00 on the first day of next
//	month
func first_of_next_month() time.Time {
	return time.Date(time.Now().Year(), time.Now().Month()+1, 1, 0, 0, 0, 0,
		time.Local)
}

// Fires an event on the first of every month, first thing in the morning
//	(00:00) at local time
func monthly_timer() *time.Timer {
	return time.NewTimer(first_of_next_month().Sub(time.Now()))
}
