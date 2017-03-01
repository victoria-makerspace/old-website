package talk

import (
	"log"
	"net/url"
	"regexp"
	"fmt"
)

func (api *Talk_api) Check_username(username string) (available bool, err string) {
	j := api.get_json("/users/check_username.json", api.admin,
		"username="+url.QueryEscape(username))
	if j, ok := j.(map[string]interface{}); ok {
		if available, ok := j["available"]; ok {
			if available.(bool) {
				return true, ""
			}
			return false, "Username not available"
		}
		if errors, ok := j["errors"]; ok {
			return false, errors.([]interface{})[0].(string)
		}
	}
	log.Panic("Talk server error during Check_username")
	return false, ""
}

type Talk_user struct {
	external_id    int
	id             int
	Username       string
	avatar_url     []byte
	Card_bg_url    string
	Profile_bg_url string
	notifications  []interface{}
	*Talk_api
}

func (api *Talk_api) parse_user(external_id int, u map[string]interface{}) *Talk_user {
	t := &Talk_user{external_id: external_id, Talk_api: api}
	t.id = int(u["id"].(float64))
	t.Username = u["username"].(string)
	t.avatar_url = []byte(u["avatar_template"].(string))
	t.Card_bg_url = api.Base_url + u["card_background"].(string)
	t.Profile_bg_url = api.Base_url + u["profile_background"].(string)
	return t
}

func (api *Talk_api) Get_user(id int) *Talk_user {
	j := api.get_json("/users/by-external/"+fmt.Sprint(id)+".json", api.admin)
	if j, ok := j.(map[string]interface{}); ok {
		if u, ok := j["user"].(map[string]interface{}); ok {
			return api.parse_user(id, u)
		}
	}
	return nil
}

var avatar_size_rexp = regexp.MustCompile("{size}")

func (t *Talk_user) Avatar_url(size int) string {
	return t.Base_url + string(avatar_size_rexp.ReplaceAll(t.avatar_url,
		[]byte(fmt.Sprint(size))))
}

/*
func (t *Talk_user) Notifications() []interface{} {
	if t.notifications != nil {
		return t.notifications
	}
	j := t.get_json("/notifications.json", t.Username)
	//TODO: check errors and parse further
	if n, ok := j.(map[string]interface{}); ok {
		if n, ok := n["notifications"].([]interface{}); ok {
			return n
		}
	}
	return nil
}*/
