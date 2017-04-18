package talk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
)

func (api *Api) Sync(external_id int, username, email, name string) (*User, error) {
	err_string := "Talk: failed to sync @" + username + " <" + email + ">"
	values := url.Values{}
	values.Set("external_id", fmt.Sprint(external_id))
	values.Set("username", username)
	values.Set("email", email)
	values.Set("name", name)
	payload, sig := api.Encode_sso_rsp(values)
	values = url.Values{}
	values.Set("sso", payload)
	values.Set("sig", sig)
	data, err := api.do_form("POST", "/admin/users/sync_sso", values)
	if err != nil {
		log.Println(err_string + ": ", err)
		return nil, fmt.Errorf("Talk server error")
	}
	if u, ok := data.(map[string]interface{}); ok {
		if e, ok := u["failed"].(string); ok {
			log.Println(err_string + ": " + e)
			return nil, fmt.Errorf(e)
		}
		user := api.parse_user(u)
		if user != nil {
			user.External_id = external_id
			return user, nil
		}
	}
	log.Println(err_string + ": ", data)
	return nil, fmt.Errorf("Talk server error")
}

func (api *Api) Parse_sso_req(q url.Values) (payload url.Values) {
	if q.Get("sso") == "" {
		return nil
	}
	payload_bytes, _ := base64.StdEncoding.DecodeString(q.Get("sso"))
	sig, _ := hex.DecodeString(q.Get("sig"))
	mac := hmac.New(sha256.New, []byte(api.Sso_secret))
	mac.Write([]byte(q.Get("sso")))
	payload, err := url.ParseQuery(string(payload_bytes))
	if err != nil || !hmac.Equal(mac.Sum(nil), sig) {
		return nil
	}
	return
}

func (api *Api) Encode_sso_rsp(q url.Values) (payload, sig string) {
	payload = base64.StdEncoding.EncodeToString([]byte(q.Encode()))
	mac := hmac.New(sha256.New, []byte(api.Sso_secret))
	mac.Write([]byte(payload))
	sig = hex.EncodeToString(mac.Sum(nil))
	payload = url.QueryEscape(payload)
	return
}

func (t *User) Logout() {
	if _, err := t.do_form("POST", "/admin/users/"+fmt.Sprint(t.Id)+"/log_out",
		nil); err != nil {
		//TODO: propagate errors
	}
}
