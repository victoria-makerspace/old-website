package talk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/url"
)

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

func (api *Talk_api) Encode_sso_rsp(nonce string, q url.Values) (payload, sig string) {
	q.Set("nonce", nonce)
	payload = base64.StdEncoding.EncodeToString([]byte(q.Encode()))
	mac := hmac.New(sha256.New, []byte(api.sso_secret))
	mac.Write([]byte(payload))
	sig = hex.EncodeToString(mac.Sum(nil))
	payload = url.QueryEscape(payload)
	return
}
