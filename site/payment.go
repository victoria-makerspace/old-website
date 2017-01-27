package site

import (
    "net/http"
    beanstream "github.com/Beanstream/beanstream-go"
    _ "github.com/Beanstream/beanstream-go/paymentMethods"
)

type Payment_api struct {
    config beanstream.Config
    gateway beanstream.Gateway
}

func Payment (merchant_id, api_key string) (b *Payment_api) {
    b = new(Payment_api)
    b.config = beanstream.Config{merchant_id, api_key, "", "", "www", "api", "v1", "-8:00"}
    b.gateway = beanstream.Gateway{b.config}
    return
}

func (s *Http_server) billing_handler () {
    s.mux.HandleFunc("/member/billing", func (w http.ResponseWriter, r *http.Request) {
s.parse_templates()
        p := page{Name: "billing", Title: "Billing"}
        s.authenticate(w, r, &p.Member)
        if p.Member.Username == "" {
            http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
            return
        }
        s.tmpl.Execute(w, p)
    })
}
