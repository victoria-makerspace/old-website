package site

import (
    _ "log"
    "net/http"
    beanstream "github.com/Beanstream/beanstream-go"
)

var billing struct {
    config beanstream.Config
    gateway beanstream.Gateway
    payments beanstream.PaymentsAPI
    profiles beanstream.ProfilesAPI
}

func Billing_setup (merchant_id, payments_api_key, profiles_api_key string) {
    billing.config = beanstream.Config{merchant_id, payments_api_key, profiles_api_key, "", "www", "api", "v1", "-8:00"}
    billing.gateway = beanstream.Gateway{billing.config}
    billing.profiles = billing.gateway.Profiles()
    return
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
        if r.PostFormValue("singleUseToken") != "" {
        }
        s.tmpl.Execute(w, p)
    })
}
