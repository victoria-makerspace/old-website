package talk

import (
	"log"
	"encoding/json"
	"net/http"
	"net/url"
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
		log.Printf("Talk access error (GET %s):\n\t%q\n", path, err)
		return nil
	}
	defer rsp.Body.Close()
	if err = json.NewDecoder(rsp.Body).Decode(&data); err != nil {
		log.Printf("Talk JSON decoding error (GET %s):\n\t%q\n", path, err)
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
	rsp, err := http.PostForm(api.Url()+path, form)
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

