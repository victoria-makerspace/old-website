package talk

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type Talk_api struct {
	Path       string
	Url        string
	admin      string
	api_key    string
	sso_secret string
}

func New_talk_api(config map[string]string) *Talk_api {
	return &Talk_api{
		Path:       config["path"],
		Url:        config["url"],
		admin:      config["admin"],
		api_key:    config["api-key"],
		sso_secret: config["sso-secret"]}
}

// First argument of query is the api_username
func (api *Talk_api) get_json(path string, use_key bool) (interface{}, error) {
	rel_path, err := url.Parse(path)
	if err != nil {
		log.Panicf("get_json input error: path = '%s'\n", path)
	}
	URL := api.Url + rel_path.EscapedPath()
	query := rel_path.Query()
	if use_key {
		query.Set("api_key", api.api_key)
		if query.Get("api_username") == "" {
			query.Set("api_username", api.admin)
		}
	}
	encoded := query.Encode()
	if encoded != "" {
		URL += "?" + encoded
	}
	rsp, err := http.Get(URL)
	if err != nil {
		return nil, fmt.Errorf("Talk HTTP protocol error (GET %s):\n\t%q\n",
			path, err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return nil, fmt.Errorf("Talk HTTP %d error (GET %s)\n",
			rsp.StatusCode, path)
	}
	var data interface{}
	if err = json.NewDecoder(rsp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("Talk JSON decoding error (GET %s):\n\t%q\n",
			path, err)
	}
	return data, nil
}

//TODO: get rid of redundancy
func (api *Talk_api) do_form(method, path string, form url.Values) (interface{}, error) {
	if form == nil {
		form = url.Values{}
	}
	form.Set("api_key", api.api_key)
	form.Set("api_username", api.admin)
	req, err := http.NewRequest(method, api.Url+path,
		strings.NewReader(form.Encode()))
	if err != nil {
		log.Panicf("do_form input error: %q\n", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Talk HTTP protocol error (%s %s):\n\t%q\n",
			method, path, err)
	}
	defer rsp.Body.Close()
	var data interface{}
	err = json.NewDecoder(rsp.Body).Decode(&data)
	if err != nil {
		if rsp.StatusCode != 200 {
			return nil, fmt.Errorf("Talk HTTP %d error (%s %s)\n",
				rsp.StatusCode, method, path)
		}
		return nil, fmt.Errorf("Talk JSON decoding error (%s %s):\n\t%q\n",
			method, path, err)
	}
	if rsp.StatusCode != 200 {
		return nil, fmt.Errorf("Talk HTTP %d error (%s %s): %q\n",
			rsp.StatusCode, method, path, data)
	}
	return data, nil
}

func (api *Talk_api) put_json(path string, form url.Values) interface{} {
	form.Set("api_key", api.api_key)
	form.Set("api_username", api.admin)
	req, err := http.NewRequest("PUT", api.Url+path,
		strings.NewReader(form.Encode()))
	if err != nil {
		log.Panic(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Talk error (PUT %s):\n\t%q\n", path, err)
		return nil
	}
	defer rsp.Body.Close()
	var data interface{}
	if err = json.NewDecoder(rsp.Body).Decode(&data); err != nil {
		if err.Error() != "EOF" {
			log.Printf("Talk JSON decoding error (PUT %s):\n\t%q\n", path, err)
		}
		return nil
	}
	return data
}

func (api *Talk_api) post_json(path string, form url.Values) interface{} {
	var data interface{}
	if form == nil {
		form = url.Values{}
	}
	form.Set("api_key", api.api_key)
	if _, ok := form["api_username"]; !ok {
		form.Set("api_username", api.admin)
	}
	rsp, err := http.PostForm(api.Url+path, form)
	if err != nil {
		log.Printf("Talk error (POST %s):\n\t%q\n", path, err)
		return nil
	}
	defer rsp.Body.Close()
	if err = json.NewDecoder(rsp.Body).Decode(&data); err != nil {
		if err.Error() != "EOF" {
			log.Printf("Talk JSON decoding error (POST %s):\n\t%q\n", path, err)
		}
		return nil
	}
	return data
}

func (api *Talk_api) Message_member(title, message string, users ...*Talk_user) {
	values := url.Values{}
	values.Set("title", title)
	values.Set("raw", message)
	values.Set("archetype", "private_message")
	usernames := users[0].Username
	for _, u := range users[1:] {
		usernames += "," + u.Username
	}
	values.Set("target_usernames", usernames)
	api.post_json("/post", values)
}

// Discourse groups as groups[name] == id
func (api *Talk_api) Groups() map[string]int {
	if data, err := api.get_json("/admin/groups.json", true); err == nil {
		if j, ok := data.([]interface{}); ok {
			groups := make(map[string]int)
			for _, group := range j {
				if g, ok := group.(map[string]interface{}); ok {
					groups[g["name"].(string)] = int(g["id"].(float64))
				}
			}
			return groups
		}
	}
	return nil
}
