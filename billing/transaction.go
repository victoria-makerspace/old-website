package billing

import (
/*	"database/sql"
	"fmt"
	beanstream "github.com/Beanstream/beanstream-go"
	"log"
	"math/rand"
	"strconv"*/
	"time"
)

// TODO: reject negative amounts

type transaction struct {
	id         string
	approved   bool
	timestamp  time.Time
	amount     float64
	order_id   string
	comment    string
	card       string // Last 4 digits
	ip_address string
	invoice    int
}

/*func (p *Profile) new_transaction(amount float64, name, ip_address string) *Transaction {
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

/*func (p *Profile) Get_transactions(number int) []*Transaction {
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
