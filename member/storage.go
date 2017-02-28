package member

import (
	"github.com/vvanpo/makerspace/billing"
	"database/sql"
	"log"
)

type Storage struct {
	*billing.Fee
	Number    int
	Size      float64
	Price     float64
	Available bool
	*Member
}

func (ms *Members) Get_storage(fee *billing.Fee) []*Storage {
	storage := make([]*Storage, 0)
	rows, err := ms.Query("SELECT number, size, invoice, available " +
		"FROM storage WHERE fee = $1 ORDER BY number ASC", fee.Id)
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			number     int
			size       sql.NullFloat64
			invoice_id sql.NullInt64
			available  bool
		)
		if err = rows.Scan(&number, &size, &invoice_id, &available);
			err != nil {
			log.Panic(err)
		}
		s := &Storage{
			Fee: fee,
			Number: number,
			Size: size.Float64,
			Price: fee.Amount,
			Available: available}
		if fee.Identifier == "wall" {
			s.Price *= s.Size
		}
		if invoice_id.Valid {
			i := ms.Get_bill(int(invoice_id.Int64))
			s.Member = ms.Get_member_by_id(i.Member)
		}
		storage = append(storage, s)
	}
	return storage
}
