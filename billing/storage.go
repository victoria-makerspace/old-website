package billing

import (
	"database/sql"
	"log"
)

type Storage map[string][]struct {
	Number int
	Size float64
	*Invoice
}

//TODO: recover from panics and send http 500 if applicable?
func (b *Billing) get_storage() {
	b.Storage = make(Storage)
	rows, err := b.db.Query("SELECT s.number, s.size, s.invoice, f.category, " +
		"f.identifier, f.description FROM storage s JOIN fee f " +
		"ON s.fee = f.id ORDER BY s.number ASC")
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			number          int
			size            sql.NullFloat64
			invoice_id      sql.NullInt64
			invoice         *Invoice
			fee_category    string
			fee_identifier  string
			fee_description string
		)
		if err = rows.Scan(&number, &size, &invoice_id, &fee_category,
			&fee_identifier, &fee_description); err != nil {
			log.Panic(err)
		}
		if invoice_id.Valid {
			invoice = b.get_bill(int(invoice_id.Int64))
		}
		key := fee_category + "_" + fee_identifier
		if _, ok := b.Fees[key]; ok {
			if _, ok := b.Storage[key]; !ok {
				b.Storage[key] = make([]struct {
					Number int
					Size float64
					*Invoice
				}, 0)
			}
			b.Storage[key] = append(b.Storage[key], struct {
				Number int
				Size float64
				*Invoice
			}{number, size.Float64, invoice})
		} else {
			log.Panicf("Storage fee '%s' not found", fee_identifier)
		}
	}
}
