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
type Storage map[string][]st

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
		//TODO: change Storage type to use *Fee instead of key
		if f := b.get_fee(fee_id); f != nil {
			key := f.Category + "_" + f.Identifier
			if _, ok := b.Storage[key]; !ok {
				b.Storage[key] = make([]st, 0)
			}
			s := st{number, size.Float64, f.Amount, available, invoice}
			if key == "storage_wall" {
				s.Price *= s.Size
			}
			b.Storage[key] = append(b.Storage[key], s)
		} else {
			log.Panicf("Storage fee '%s' not found", f.Identifier)
		}
	}
}
