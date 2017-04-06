package talk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
)

func (api *Talk_api) Sync(external_id int, username, email, name string) *Talk_user {
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
	//TODO: propagate errors
	if err != nil {
		return nil
	}
	if u, ok := data.(map[string]interface{}); ok {
		if _, ok := u["failed"]; ok {
			return nil
		}
		return api.parse_user(external_id, u)
	}
	return nil
}

func (api *Talk_api) Parse_sso_req(q url.Values) (payload url.Values) {
	if q.Get("sso") == "" {
		return nil
	}
	payload_bytes, _ := base64.StdEncoding.DecodeString(q.Get("sso"))
	sig, _ := hex.DecodeString(q.Get("sig"))
	mac := hmac.New(sha256.New, []byte(api.sso_secret))
	mac.Write([]byte(q.Get("sso")))
	payload, err := url.ParseQuery(string(payload_bytes))
	if err != nil || !hmac.Equal(mac.Sum(nil), sig) {
		return nil
	}
	return
}

func (api *Talk_api) Encode_sso_rsp(q url.Values) (payload, sig string) {
	payload = base64.StdEncoding.EncodeToString([]byte(q.Encode()))
	mac := hmac.New(sha256.New, []byte(api.sso_secret))
	mac.Write([]byte(payload))
	sig = hex.EncodeToString(mac.Sum(nil))
	payload = url.QueryEscape(payload)
	return
}

func (t *Talk_user) Logout() {
	t.post_json("/admin/users/"+fmt.Sprint(t.id)+"/log_out", nil)
}
