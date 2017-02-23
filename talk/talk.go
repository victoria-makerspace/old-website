package talk

import (
	"fmt"
	"encoding/json"
	"net/http"
	"net/url"
	"log"
	"regexp"
)

type Talk_api struct {
	Url string
	admin string
	api_key string
	sso_secret string
}

func New_talk_api (url, admin, api_key, sso_secret string) *Talk_api {
	return &Talk_api{
		Url: url,
		admin: admin,
		api_key: api_key,
		sso_secret: sso_secret}
}

func (api *Talk_api) get_json(path, username string) interface{} {
	var data interface{}
	//TODO: perhaps parse as url.URL first in case parameters have already been
	//	added.
	rsp, err := http.Get(api.Url + path + "?api_key=" + api.api_key +
		"&api_username=" + username)
	if err != nil {
		log.Printf("Talk access error (%s):\n\t%q\n", path, err)
		return nil
	}
	defer rsp.Body.Close()
	if err = json.NewDecoder(rsp.Body).Decode(&data); err != nil {
		log.Printf("Talk JSON decoding error (%s):\n\t%q\n", path, err)
		return nil
	}
	return data
}

func (api *Talk_api) post(path string) {
	v := url.Values{}
	v.Set("api_key", api.api_key)
	v.Set("api_username", api.admin)
	rsp, err := http.PostForm(api.Url + path, v)
	if err != nil {
		log.Println(err)
		return
	}
	rsp.Body.Close()
}

type Talk_user struct {
	external_id int
	id int
	Username string
	avatar_url []byte
	Card_bg_url string
	Profile_bg_url string
	notifications []interface{}
	*Talk_api
}

func (api *Talk_api) Get_user(id int) *Talk_user {
	t := &Talk_user{external_id: id, Talk_api: api}
	j := t.get_json("/users/by-external/" + fmt.Sprint(id) + ".json", t.admin)
	if j, ok := j.(map[string]interface{}); ok {
		if u, ok := j["user"].(map[string]interface{}); ok {
			t.id = int(u["id"].(float64))
			t.Username = u["username"].(string)
			t.avatar_url = []byte(u["avatar_template"].(string))
			t.Card_bg_url = u["card_background"].(string)
			t.Profile_bg_url = u["profile_background"].(string)
			return t
		}
	}
	return nil
}

var avatar_size_rexp = regexp.MustCompile("{size}")
func (t *Talk_user) Avatar_url(size int) string {
	return string(avatar_size_rexp.ReplaceAll(t.avatar_url,
		[]byte(fmt.Sprint(size))))
}

func (t *Talk_user) Logout() {
	t.post("/admin/users/" + fmt.Sprint(t.id) + "/log_out")
}

func (t *Talk_user) Sync() {
}

func (t *Talk_user) Notifications() []interface{} {
	if t.notifications != nil {
		return t.notifications
	}
	j := t.get_json("/notifications.json", t.Username)
	//TODO: check errors and parse further
	if n, ok := j.(map[string]interface{}); ok {
		return n["notifications"].([]interface{})
	}
	return nil
}
