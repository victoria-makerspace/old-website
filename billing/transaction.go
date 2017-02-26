package billing

import (
	"database/sql"
	"fmt"
	beanstream "github.com/Beanstream/beanstream-go"
	"log"
	"math/rand"
	"strconv"
	"time"
)

const minimum_txn_amount = 1.00

type Transaction struct {
	Id int
	*Profile
	Approved   bool
	Time       time.Time
	Amount     float64
	Comment    string
	Card       string // Last 4 digits
	Ip_address string //TODO
	Invoice    *Invoice
	order_id   string
}

func (p *Profile) do_transaction(amount float64, comment string, invoice *Invoice) *Transaction {
	if amount < minimum_txn_amount {
		log.Printf("Transaction for member %d below minimum amount (%f < %f)",
			p.member_id, amount, minimum_txn_amount)
		return nil
	}
	if p.Error != None {
		//TODO: missed payment
		return nil
	}
	order_id := fmt.Sprintf("%d-%d", rand.Intn(1000000), p.member_id)
	txn := &Transaction{
		Profile:  p,
		Time:     time.Now(),
		Amount:   amount,
		Comment:  comment,
		Invoice:  invoice,
		order_id: order_id}
	req := beanstream.PaymentRequest{
		PaymentMethod: "payment_profile",
		OrderNumber:   order_id,
		Amount:        float32(amount),
		Profile:       beanstream.ProfilePayment{p.bs_id, 1, true},
		Comment:       comment}
	rsp, err := p.payment_api.MakePayment(req)
	if err != nil {
		//TODO: log missed payment
		log.Println(err)
		return nil
	}
	if !rsp.IsApproved() {
		//TODO: log missed payment
		//TODO: make sure unapproved == invalid card
		if p.Error != None {
			p.set_error(Invalid_card)
		}
		log.Println("Payment of %.2f by member %d failed", amount, p.member_id)
	} else {
		p.clear_error()
	}
	txn.Id, _ = strconv.Atoi(rsp.ID)
	txn.Approved = rsp.IsApproved()
	txn.Card = rsp.Card.LastFour
	approved := "approved"
	if !txn.Approved {
		approved = "failed (" + fmt.Sprint(rsp.Message) + ")"
	}
	log.Printf("Transaction %d for member %d %s", txn.Id, p.member_id, approved)
	if _, err := p.db.Exec("INSERT INTO transaction "+
		"(id, profile, approved, time, amount, order_id, comment, card, "+
		"invoice) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		txn.Id, p.member_id, txn.Approved, txn.Time, txn.Amount,
		txn.order_id, txn.Comment, txn.Card, txn.Invoice.Id); err != nil {
		log.Panic(err)
	}
	return txn
}

func (p *Profile) do_recurring_txn(i *Invoice) *Transaction {
	return p.do_transaction(i.Amount, i.Description, i)
}

func (t *Transaction) log_recurring_txn(log_id int) {
	if _, err := t.db.Exec(
		"UPDATE transaction "+
			"SET logged = $1 "+
			"WHERE id = $2",
		log_id, t.Id); err != nil {
		log.Panic(err)
	}
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
	rows, err := p.db.Query("SELECT id, approved, time, amount, "+
		"order_id, comment, card, ip_address, invoice FROM transaction WHERE "+
		"profile = $1 ORDER BY time DESC", p.member_id)
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
