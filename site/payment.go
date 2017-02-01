package site

import (
    "database/sql"
    "log"
    "net/http"
    "time"
    beanstream "github.com/Beanstream/beanstream-go"
    "github.com/lib/pq"
)

type Billing struct {
    Card_number string
    Card_expiry string
    Student bool
    Student_institution string
    Student_graduation_date time.Time
}

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

func (s *Http_server) billing_create_profile (token, name, username string) {
    req := beanstream.Profile{
        Token: beanstream.Token{
            Token: token,
            Name: name},
        Custom: beanstream.CustomFields{Ref1: username}}
    rsp, _ := profiles.CreateProfile(req)
    id := rsp.Id
    log.Println(id)
    log.Println(rsp.Code)
    log.Println(rsp.Message)
    _, err := s.db.Exec("INSERT INTO billing_profile VALUES ($1, $2)", username, id)
    if err != nil { log.Panic(err) }
}

func (s *Http_server) billing_handler () {
    s.mux.HandleFunc("/member/billing", func (w http.ResponseWriter, r *http.Request) {
s.parse_templates()
        p := page{Name: "billing", Title: "Billing"}
        s.authenticate(w, r, &p.Member)
        if !p.Member.Authenticated() {
            http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
            return
        }
        var id string
        err := s.db.QueryRow("SELECT id FROM billing_profile WHERE username = $1", p.Member.Username).Scan(&id)
        if err == nil {
            profile, _ := profiles.GetProfile(id)
            if profile != nil {
                p.Member.Billing.Card_number = profile.Card.Number
                p.Member.Billing.Card_expiry = profile.Card.ExpiryMonth + "/20" + profile.Card.ExpiryYear
            } else {
                log.Println("missing beanstream profile " + id)
                id = ""
            }
        } else if err != sql.ErrNoRows && err != nil { log.Panic(err) }
        if token := r.PostFormValue("singleUseToken"); token != "" {
            if id != "" {
                rsp, _ := profiles.DeleteCard(id, 1)
                log.Println(rsp)
                rsp, _ = profiles.AddTokenizedCard(id, r.PostFormValue("name"), token)
                log.Println(rsp)
            } else {
                s.billing_create_profile(token, p.Member.Name, p.Member.Username)
            }
        }
        var (
            institution sql.NullString
            graduation_date pq.NullTime
        )
        err = s.db.QueryRow("SELECT institution, graduation_date FROM student WHERE username = $1", p.Member.Username).Scan(&institution, &graduation_date)
        if err != nil {
            if err != sql.ErrNoRows { log.Panic(err) }
        } else {
            p.Member.Billing.Student = true
            p.Member.Billing.Student_institution = institution.String
            p.Member.Billing.Student_graduation_date = graduation_date.Time
        }
        s.tmpl.Execute(w, p)
    })
}
