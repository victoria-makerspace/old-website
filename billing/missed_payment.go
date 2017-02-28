package billing

import (
	"log"
	"time"
	"database/sql"
)

type Missed_payment struct {
	*Invoice
	time.Time
	Txn_id int
	log_id int
}

func (p *Profile) do_missed_payment(i *Invoice, txn *Transaction) {
	log.Printf("Payment of $%.2f by member %d failed\n", i.Amount, p.member_id)
	if txn != nil {
		if _, err := p.db.Exec(
			"INSERT INTO missed_payment (invoice, transaction) "+
			"VALUES ($1, $2)", i.Id, txn.Id); err != nil {
			log.Panic(err)
		}
		return
	}
	if _, err := p.db.Exec(
		"INSERT INTO missed_payment (invoice) "+
		"VALUES ($1)", i.Id); err != nil {
		log.Panic(err)
	}
}

func (p *Profile) get_missed_payment(i *Invoice) *Missed_payment {
	mp := &Missed_payment{Invoice: i}
	var txn_id, log_id sql.NullInt64
	if err := p.db.QueryRow(
		"SELECT time, transaction, logged "+
		"FROM missed_payment "+
		"WHERE invoice = $1", i.Id).Scan(&mp.Time, &txn_id, &log_id);
		err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	mp.Txn_id = int(txn_id.Int64)
	mp.log_id = int(log_id.Int64)
	return mp
}

func (p *Profile) log_missed_payment(mp *Missed_payment, log_id int) {
	if _, err := p.db.Exec(
		"UPDATE missed_payment "+
		"SET logged = $3 "+
		"WHERE invoice = $1 AND time = $2",
		mp.Invoice.Id, mp.Time, log_id); err != nil {
		log.Panic(err)
	}
}
