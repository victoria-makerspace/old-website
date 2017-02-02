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

func (s *Http_server) billing_create_profile (token, name, username string) (id string) {
    req := beanstream.Profile{
        Token: beanstream.Token{
            Token: token,
            Name: name},
        Custom: beanstream.CustomFields{Ref1: username}}
    rsp, err := profiles.CreateProfile(req)
    if err != nil { log.Panic(err) }
    id = rsp.Id
    _, err = s.db.Exec("INSERT INTO billing_profile VALUES ($1, $2)", username, id)
    if err != nil { log.Panic(err) }
    return
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
        fetch_profile := func (id string) {
            if profile, err := profiles.GetProfile(id); err != nil {
                //berr := err.(*beanstream.BeanstreamApiException)
                log.Panic(err)
            } else {
                p.Member.Billing.Card_number = profile.Card.Number
                p.Member.Billing.Card_expiry = profile.Card.ExpiryMonth + "/20" + profile.Card.ExpiryYear
            }
        }
        var id string
        err := s.db.QueryRow("SELECT id FROM billing_profile WHERE username = $1", p.Member.Username).Scan(&id)
        if err == nil {
            fetch_profile(id)
        } else if err != sql.ErrNoRows && err != nil {
            log.Panic(err)
        }
        if token := r.PostFormValue("singleUseToken"); token != "" {
            if id != "" {
                if _, err = profiles.DeleteCard(id, 1); err != nil { log.Panic(err) }
                if _, err = profiles.AddTokenizedCard(id, r.PostFormValue("name"), token); err != nil { log.Panic(err) }
            } else {
                fetch_profile(s.billing_create_profile(token, p.Member.Name, p.Member.Username))
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
        if r.PostFormValue("billing") == "true" {
            if r.PostFormValue("rate") == "student" && r.PostFormValue("institution") != "" && r.PostFormValue("graduation") != "" {
                graduation, err := time.Parse("2006-01", r.PostFormValue("graduation"))
                if err == nil && graduation.After(time.Now().AddDate(0, 1, 0)) {
                    if p.Member.Billing.Student {
                        _, err = s.db.Exec("UPDATE student SET institution = $2, graduation_date = $3 WHERE username = $1", p.Member.Username, r.PostFormValue("institution"), graduation)
                    } else {
                        _, err = s.db.Exec("INSERT INTO student (username, institution, graduation_date) VALUES ($1, $2, $3)", p.Member.Username, r.PostFormValue("institution"), graduation)
                    }
                    if err != nil { log.Panic(err) }
                    p.Member.Billing.Student = true
                    p.Member.Billing.Student_institution = r.PostFormValue("institution")
                    p.Member.Billing.Student_graduation_date = graduation
                }
            } else if p.Member.Billing.Student && r.PostFormValue("rate") == "regular" {
                _, err = s.db.Exec("DELETE FROM student WHERE username = $1", p.Member.Username)
                if err != nil { log.Panic(err) }
                p.Member.Billing.Student = false
                p.Member.Billing.Student_institution = ""
                p.Member.Billing.Student_graduation_date = time.Unix(0, 0)
            }
        }
        s.tmpl.Execute(w, p)
    })
}
