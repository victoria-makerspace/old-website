package billing

import (
	"database/sql"
	//beanstream "github.com/Beanstream/beanstream-go"
	"log"
	"time"
)

type Transaction struct {
	Id int
	*Profile
	Approved   bool
	Time       time.Time
	Amount     float64
	Comment    string
	Card       string // Last 4 digits
	Ip_address string
	Invoice    *Invoice
	order_id   string
}

/*func (p *Profile) new_transaction(amount float64, name, ip_address string) *Transaction {
// TODO: reject negative amounts

	if amount <= 0 {
		return nil
	}
	order_id := fmt.Sprint(rand.Intn(1000000)) + "-" + p.member.Username
	req := beanstream.PaymentRequest{
		PaymentMethod: "payment_profile",
		OrderNumber:   order_id,
		Amount:        float32(amount),
		Profile:       beanstream.ProfilePayment{p.Id, 1, true},
		Comment:       name,
		CustomerIp:    ip_address,
	}
	rsp, err := p.payments.MakePayment(req)
	if err != nil {
		log.Println(err)
	}
	if !rsp.IsApproved() {
		log.Println("Payment of %.2f by %s failed", amount, p.member.Username)
	}
	txn := &Transaction{id: rsp.ID,
		username:   p.member.Username,
		Date:       time.Now(),
		Approved:   rsp.IsApproved(),
		Order_id:   rsp.OrderNumber,
		Amount:     amount,
		Name:       name,
		Card:       rsp.Card.LastFour,
		Ip_address: ip_address}
	_, err = p.db.Exec("INSERT INTO transaction (id, username, approved, order_id, amount, name, card, ip_address) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", rsp.ID, p.member.Username, txn.Approved, txn.Order_id, txn.Amount, txn.Name, txn.Card, txn.Ip_address)
	if err != nil {
		log.Panic(err)
	}
	return txn
}*/

func (p *Profile) get_transactions() {
	rows, err := p.db.Query("SELECT id, approved, time, amount, order_id, "+
		"comment, card, ip_address, invoice FROM transaction WHERE "+
		"profile = $1 ORDER BY time DESC", p.member.Username)
	defer rows.Close()
	if err != nil {
		if err == sql.ErrNoRows {
			return
		}
		log.Panic(err)
	}
	for rows.Next() {
		txn := &Transaction{Profile: p}
		var order_id, comment, card, ip_address sql.NullString
		var invoice_id sql.NullInt64
		if err = rows.Scan(&txn.Id, &txn.Approved, &txn.Time, &txn.Amount,
			&order_id, &comment, &card, &ip_address, &invoice_id); err != nil {
			log.Panic(err)
		}
		txn.order_id = order_id.String
		txn.Comment = comment.String
		txn.Card = card.String
		txn.Ip_address = ip_address.String
		if invoice_id.Valid {
			txn.Invoice = p.Get_bill(int(invoice_id.Int64))
		}
		p.Transactions = append(p.Transactions, txn)
	}
}

//	Get_transaction returns nil when the transaction isn't found.
func (p *Profile) Get_transaction(id int) *Transaction {
	for _, i := range p.Transactions {
		if i.Id == id {
			return i
		}
	}
	return nil
}

/*
	var txns []*Transaction
	rows, err := p.db.Query("SELECT id, approved, order_id, amount, name, card, ip_address, time FROM transaction WHERE username = $1 ORDER BY time DESC LIMIT $2", p.member.Username, number)
	defer rows.Close()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return txns
	}
	for i := 0; rows.Next(); i++ {
		txn := &Transaction{username: p.member.Username}
		txns = append(txns, txn)
		var (
			amount     string
			name       sql.NullString
			card       sql.NullString
			ip_address sql.NullString
		)
		if err := rows.Scan(&txn.id, &txn.Approved, &txn.Order_id, &amount, &name, &card, &ip_address, &txn.Date); err != nil {
			log.Panic(err)
		}
		if txn.Amount, err = strconv.ParseFloat(amount[1:], 32); err != nil {
			log.Println(err)
		}
		txn.Name = name.String
		txn.Card = card.String
		txn.Ip_address = ip_address.String
	}
	if err := rows.Err(); err != nil {
		log.Panic(err)
	}
	return txns
}*/
