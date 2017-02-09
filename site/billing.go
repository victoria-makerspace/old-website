package site

import (
	"database/sql"
	"github.com/lib/pq"
	"log"
	"net/http"
	"strings"
	"time"
)

func (m *member) update_membership_rate(db *sql.DB) {
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM student WHERE username = $1", m.Username).Scan(&n)
	if err != nil {
		log.Panic(err)
	}
	if m.Student != nil {
		if n == 1 {
			m.Student = &student{}
			_, err = db.Exec("UPDATE student SET institution = $2, graduation_date = $3 WHERE username = $1", m.Username, m.Student.Institution, m.Student.Grad_date)
		} else {
			_, err = db.Exec("INSERT INTO student (username, institution, graduation_date) VALUES ($1, $2, $3)", m.Username, m.Student.Institution, m.Student.Grad_date)
		}
		if err != nil {
			log.Panic(err)
		}
		return
	} else if n == 1 {
		_, err = db.Exec("DELETE FROM student WHERE username = $1", m.Username)
		if err != nil {
			log.Panic(err)
		}
	}
}

/*
// http://support.beanstream.com/bic/w/docs/creating-account-via-api.htm
func (m *member) recurring_billing() {
	//var responseType map[string]interface{}
	order_id := fmt.Sprint(rand.Intn(1000000)) + "-" + m.Username
	start_date := fmt.Sprintf("%02d01%4d", time.Now().Month()+1, time.Now().Year())
	data := "?requestType=BACKEND" +
			"&merchant_id=" + config.MerchantId +
			"&customerCode=" + m.Billing.Profile_id +
			"&trnOrderNumber=" + order_id +
			"&trnAmount=50" +
			"&trnRecurring=1" +
			"&rbBillingPeriod=M" +
			"&rbBillingIncrement=1" +
			"&rbCharge=0" +
			"&rbSecondBilling" + start_date +
			"&ref1" + url.QueryEscape(m.Username)
	rsp, err := beanstream.Process("GET", "https://www.beanstream.com/scripts/process_transaction.asp" + data, config.MerchantId, config.PaymentsApiKey, responseType)
	if err != nil {
		log.Println(err)
	}
	log.Println(rsp)
	req, err := http.NewRequest("GET", "https://www.beanstream.com/scripts/process_transaction.asp" + data, nil)
	//req, err := http.NewRequest("GET", "https://www.beanstream.com/api/v1/scripts" + data, nil)
	req.Header.Set("Authorization", "Passcode "+beanstream.GenerateAuthCode(config.MerchantId, "432A1D461BDa47d48A2b285F5D254902"))//config.ProfilesApiKey))
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Panic(err)
	}
	defer rsp.Body.Close()
	body, _ := ioutil.ReadAll(rsp.Body)
	log.Println(url.QueryUnescape(string(body)))
}*/

func (s *Http_server) billing_handler() {
	s.mux.HandleFunc("/member/billing", func(w http.ResponseWriter, r *http.Request) {
		s.parse_templates()
		p := page{Name: "billing", Title: "Billing"}
		s.authenticate(w, r, &p.Member)
		if !p.Member.Authenticated() {
			http.Error(w, http.StatusText(403), 403)
			return
		}
		p.Member.Billing = s.billing.Get_profile(p.Member.Username)
		r.ParseForm()
		if token := r.PostFormValue("singleUseToken"); token != "" {
			if p.Member.Billing != nil {
				p.Member.Billing.Update_card(r.PostFormValue("name"), token)
			} else {
				p.Member.Billing = s.billing.New_profile(token, p.Member.Name, p.Member.Username)
			}
			http.Redirect(w, r, "/member/billing", 303)
		} else if _, ok := r.PostForm["delete-card"]; ok && p.Member.Billing != nil {
			p.Member.Billing.Delete_card()
		}
		if _, ok := r.PostForm["register"]; ok {
			if r.PostFormValue("rate") == "student" && r.PostFormValue("institution") != "" && r.PostFormValue("graduation") != "" {
				graduation, err := time.Parse("2006-01", r.PostFormValue("graduation"))
				if err == nil && graduation.After(time.Now().AddDate(0, 1, 0)) {
					p.Member.Student = &student{r.PostFormValue("institution"), graduation}
				}
			}
			p.Member.update_membership_rate(s.db)
			ip := r.RemoteAddr[:strings.LastIndexByte(r.RemoteAddr, ':')]
			p.Member.Billing.New_transaction(50, "Membership dues", ip)
			http.Redirect(w, r, "/member/billing", 303)
		} else {
			var (
				institution sql.NullString
				grad_date   pq.NullTime
			)
			err := s.db.QueryRow("SELECT institution, graduation_date FROM student WHERE username = $1", p.Member.Username).Scan(&institution, &grad_date)
			if err != nil {
				if err != sql.ErrNoRows {
					log.Panic(err)
				}
			} else {
				p.Member.Student = &student{institution.String, grad_date.Time}
			}
		}
		s.tmpl.Execute(w, p)
	})
}
