package site

import (
    _ "log"
    "net/http"
    beanstream "github.com/Beanstream/beanstream-go"
)

var (
    config beanstream.Config
    gateway beanstream.Gateway
    payments beanstream.PaymentsAPI
    profiles beanstream.ProfilesAPI
)

func Billing_setup (b map[string]string) {
    config = beanstream.Config{b["merchant-id"], b["payments-api-key"], b["profiles-api-key"], "", "www", "api", "v1", "-8:00"}
    gateway = beanstream.Gateway{config}
    profiles = gateway.Profiles()
}

func (s *Http_server) billing_handler () {
    s.mux.HandleFunc("/member/billing", func (w http.ResponseWriter, r *http.Request) {
s.parse_templates()
        p := page{Name: "billing", Title: "Billing"}
        s.authenticate(w, r, &p.Member)
        if !p.Authenticated() {
            http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
            return
        }
        profile, _ := profiles.GetProfile(p.Member.Username)
        if profile != nil {
            p.Member.Billing.Card_number = profile.Card.Number
            p.Member.Billing.Card_expiry = profile.Card.ExpiryMonth + "/20" + profile.Card.ExpiryYear
        }
        if token := r.PostFormValue("singleUseToken"); token != "" {
        }
        s.tmpl.Execute(w, p)
    })
}
