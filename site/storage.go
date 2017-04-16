package site

import (
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/sub"
	"strconv"
)

func init() {
	init_handler("storage", storage_handler, "/member/storage")
}

func storage_handler(p *page) {
	p.Title = "Storage"
	if !p.must_authenticate() {
		return
	}
	//TODO: move this into member/storage.go
	if plan := p.PostFormValue("register-storage-plan"); plan != "" {
		if !p.Has_card() {
			p.http_error(403)
			return
		}
		var quantity uint64
		var valid bool
		number, err := strconv.Atoi(p.PostFormValue("register-storage-number"))
		if err != nil {
			p.http_error(400)
			return
		}
		for _, s := range p.List_storage(plan) {
			if s.Number == number {
				if s.Member != nil || !s.Available {
					p.http_error(403)
					return
				}
				quantity = s.Quantity
				valid = true
				break
			}
		}
		if !valid {
			p.http_error(422)
			return
		}
		params := &stripe.SubParams{
			Customer: p.Customer().ID,
			Plan: plan,
			Quantity: quantity}
		params.Params.Meta = make(map[string]string)
		params.Meta["number"] = p.PostFormValue("register-storage-number")
		sub.New(params)
	}
}
