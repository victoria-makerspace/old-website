package billing

import (
	"database/sql"
	"log"
	"time"
)

type Missed_payment struct {
	*Invoice
	time.Time
	Txn_id int
	log_id int
}

func (p *Profile) do_missed_payment(i *Invoice, txn *Transaction) *Missed_payment {
	log.Printf("Payment of $%.2f by member %d failed\n", i.Amount, p.member_id)
	mp := &Missed_payment{Invoice: i}
	if txn != nil {
		mp.Txn_id = txn.Id
		if err := p.db.QueryRow(
			"INSERT INTO missed_payment (invoice, transaction) "+
				"VALUES ($1, $2) "+
				"RETURNING time", i.Id, txn.Id).Scan(&mp.Time); err != nil {
			log.Panic(err)
		}
	} else if err := p.db.QueryRow(
		"INSERT INTO missed_payment (invoice) "+
			"VALUES ($1) "+
			"RETURNING time", i.Id).Scan(&mp.Time); err != nil {
		log.Panic(err)
	}
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

func (p *Profile) delete_missed_payment(mp *Missed_payment) {
	if _, err := p.db.Exec(
		"DELETE FROM missed_payment "+
			"WHERE invoice = $1 AND time = $2",
		mp.Invoice.Id, mp.Time); err != nil {
		log.Panic(err)
	}
}

func (p *Profile) Get_missed_payments() []*Missed_payment {
	mps := make([]*Missed_payment, 0)
	rows, err := p.db.Query(
		"SELECT mp.invoice, mp.time, mp.transaction, mp.logged "+
			"FROM missed_payment mp "+
			"JOIN invoice i "+
			"ON i.id = mp.invoice "+
			"WHERE i.member = $1 "+
			"ORDER BY mp.time DESC", p.member_id)
	defer rows.Close()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Panic(err)
	}
	for rows.Next() {
		mp := &Missed_payment{}
		var invoice_id int
		var txn_id, log_id sql.NullInt64
		if err := rows.Scan(&invoice_id, &mp.Time, &txn_id, &log_id); err != nil {
			log.Panic(err)
		}
		if mp.Invoice = p.Get_bill(invoice_id); mp.Invoice == nil {
			log.Panicf("Failed to find invoice %d for missed payment (member %d)\n",
				invoice_id, p.member_id)
		}
		mp.Txn_id = int(txn_id.Int64)
		mp.log_id = int(log_id.Int64)
		mps = append(mps, mp)
	}
	return mps
}

func (p *Profile) Retry_missed_payments() {
	for _, mp := range p.Get_missed_payments() {
		p.Error = None
		if txn := p.do_transaction(mp.Invoice); txn != nil && txn.Approved {
			p.delete_missed_payment(mp)
		}
	}
}
