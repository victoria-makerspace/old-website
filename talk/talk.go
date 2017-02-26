package talk

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
)

type Talk_api struct {
	Base_url   string
	Path       string
	admin      string
	api_key    string
	sso_secret string
}

func New_talk_api(config map[string]string) *Talk_api {
	return &Talk_api{
		Base_url:   config["base-url"],
		Path:       config["path"],
		admin:      config["admin"],
		api_key:    config["api-key"],
		sso_secret: config["sso-secret"]}
}

func (api *Talk_api) Url() string {
	return api.Base_url + api.Path
}

// First argument of query is the api_username
func (api *Talk_api) get_json(path string, query ...string) interface{} {
	var data interface{}
	//TODO: perhaps parse as url.URL first in case parameters have already been
	//	added.
	url := api.Url() + path + "?api_key=" + api.api_key
	if len(query) > 0 && query[0] != "" {
		url += "&api_username=" + query[0]
		for _, q := range query[1:] {
			url += "&" + q
		}
	}
	rsp, err := http.Get(url)
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

func (api *Talk_api) post(path string, form url.Values) {
	form.Set("api_key", api.api_key)
	if _, ok := form["api_username"]; !ok {
		form.Set("api_username", api.admin)
	}
	rsp, err := http.PostForm(api.Url()+path, form)
	if err != nil {
		log.Println(err)
		return
	}
	rsp.Body.Close()
}

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
			return false, errors.([]string)[0]
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

func (api *Talk_api) Get_user(id int) *Talk_user {
	t := &Talk_user{external_id: id, Talk_api: api}
	j := t.get_json("/users/by-external/"+fmt.Sprint(id)+".json", t.admin)
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
