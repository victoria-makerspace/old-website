package billing

import (
	"database/sql"
	"log"
)

type st struct {
	Number    int
	Size      float64
	Price     float64
	Available bool
	*Invoice
}
type Storage map[*Fee][]st

//TODO: recover from panics and send http 500 if applicable?
func (b *Billing) get_storage() {
	b.Storage = make(Storage)
	rows, err := b.db.Query("SELECT number, size, invoice, fee, available " +
		"FROM storage ORDER BY number ASC")
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			number     int
			size       sql.NullFloat64
			invoice_id sql.NullInt64
			invoice    *Invoice
			fee_id     int
			available  bool
		)
		if err = rows.Scan(&number, &size, &invoice_id, &fee_id, &available); err != nil {
			log.Panic(err)
		}
		if invoice_id.Valid {
			invoice = b.get_bill(int(invoice_id.Int64))
		}
		f, ok := b.Fees[fee_id]
		if !ok {
			log.Panicf("Storage fee '%s' not found", f.Identifier)
		}
		if _, ok := b.Storage[f]; !ok {
			b.Storage[f] = make([]st, 0)
		}
		s := st{number, size.Float64, f.Amount, available, invoice}
		if f.Identifier == "wall" {
			s.Price *= s.Size
		}
		b.Storage[f] = append(b.Storage[f], s)
	}
}
