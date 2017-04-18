package talk

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type Api struct {
	Path       string
	Url        string
	Admin      string
	Api_key    string
	Sso_secret string
}

// First argument of query is the api_username
func (api *Api) get_json(path string, use_key bool) (interface{}, error) {
	rel_path, err := url.Parse(path)
	if err != nil {
		log.Panicf("get_json input error: path = '%s'\n", path)
	}
	URL := api.Url + rel_path.EscapedPath()
	query := rel_path.Query()
	if use_key {
		query.Set("api_key", api.Api_key)
		if query.Get("api_username") == "" {
			query.Set("api_username", api.Admin)
		}
	}
	encoded := query.Encode()
	if encoded != "" {
		URL += "?" + encoded
	}
	rsp, err := http.Get(URL)
	if err != nil {
		return nil, fmt.Errorf("Talk HTTP protocol error (GET %s):\n\t%q",
			path, err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return nil, fmt.Errorf("Talk HTTP %d error (GET %s)",
			rsp.StatusCode, path)
	}
	var data interface{}
	if err = json.NewDecoder(rsp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("Talk JSON decoding error (GET %s):\n\t%q",
			path, err)
	}
	return data, nil
}

//TODO: get rid of redundancy
func (api *Api) do_form(method, path string, form url.Values) (interface{}, error) {
	if form == nil {
		form = url.Values{}
	}
	form.Set("api_key", api.Api_key)
	form.Set("api_username", api.Admin)
	req, err := http.NewRequest(method, api.Url+path,
		strings.NewReader(form.Encode()))
	if err != nil {
		log.Panicf("do_form input error: %q", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Talk HTTP protocol error (%s %s):\n\t%q",
			method, path, err)
	}
	defer rsp.Body.Close()
	var data interface{}
	err = json.NewDecoder(rsp.Body).Decode(&data)
	if err != nil {
		if rsp.StatusCode != 200 {
			return nil, fmt.Errorf("Talk HTTP %d error (%s %s)",
				rsp.StatusCode, method, path)
		}
		return nil, fmt.Errorf("Talk JSON decoding error (%s %s):\n\t%q",
			method, path, err)
	}
	if rsp.StatusCode != 200 {
		return nil, fmt.Errorf("Talk HTTP %d error (%s %s): %q",
			rsp.StatusCode, method, path, data)
	}
	return data, nil
}

func (api *Api) put_json(path string, form url.Values) interface{} {
	form.Set("api_key", api.Api_key)
	form.Set("api_username", api.Admin)
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

func (api *Api) Message_user(title, message string, users ...*User) {
	values := url.Values{}
	values.Set("title", title)
	values.Set("raw", message)
	values.Set("archetype", "private_message")
	usernames := users[0].Username
	for _, u := range users[1:] {
		usernames += "," + u.Username
	}
	values.Set("target_usernames", usernames)
	if _, err := api.do_form("POST", "/posts.json", values); err != nil {
		//TODO: propagate errors
	}
}

// Discourse groups as groups[name] == id
func (api *Api) All_groups() map[string]int {
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

func (api *Api) Add_to_group(group string, users ...*User) error {
	gid, ok := api.All_groups()[group]
	if !ok {
		return fmt.Errorf("'%s' is not a valid group", group)
	}
	usernames := make([]string, 0)
	for _, t := range users {
		if _, ok := t.Groups[group]; !ok {
			usernames = append(usernames, url.QueryEscape(t.Username))
		}
	}
	if len(usernames) == 0 {
		return nil
	}
	form := url.Values{}
	form.Add("usernames", strings.Join(usernames, ","))
	data, err := api.do_form("PUT", "/groups/"+fmt.Sprint(gid)+"/members.json",
		form)
	if err != nil {
		return fmt.Errorf("Error adding to Talk group '%s': %q\n", group, err)
	}
	if j, ok := data.(map[string]interface{}); ok {
		if _, ok := j["success"]; ok {
			return nil
		}
	}
	return fmt.Errorf("Error adding to Talk group '%s': %q\n", group, data)
}
