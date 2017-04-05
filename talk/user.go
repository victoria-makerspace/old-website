package talk

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func (api *Talk_api) Check_username(username string) (available bool, err string) {
	j, e := api.get_json("/users/check_username.json?username="+
		url.QueryEscape(username), false)
	if e != nil {
		return false, "Server error"
	}
	if j, ok := j.(map[string]interface{}); ok {
		if errors, ok := j["errors"]; ok {
			return false, "Username " + errors.([]interface{})[0].(string)
		} else if _, ok := j["available"]; ok {
			// Even if talk gives available = false, (e.g. for staged users),
			//	sso automatically merges the talk user instance with the
			//	newly-created local one, hence we still return 'true'
			return true, ""
		}
		// Unanticipated JSON response
		log.Printf("Talk server rejected username '%s': %q\n", username, j)
		return false, "Username not available"
	}
	log.Panicf("Talk server parsing error during Check_username: %s\n",
		username)
	return
}

type Talk_user struct {
	external_id    int
	id             int
	Username       string
	avatar_url     []byte
	Card_bg_url    string
	Profile_bg_url string
	*Talk_api
}

func (api *Talk_api) parse_user(external_id int, u map[string]interface{}) *Talk_user {
	t := &Talk_user{external_id: external_id, Talk_api: api}
	t.id = int(u["id"].(float64))
	t.Username = u["username"].(string)
	t.avatar_url = []byte(u["avatar_template"].(string))
	if card, ok := u["card_background"].(string); ok {
		t.Card_bg_url = card
	}
	if profile, ok := u["profile_background"].(string); ok {
		t.Profile_bg_url = profile
	}
	return t
}

func (api *Talk_api) Get_user(external_id int) *Talk_user {
	j, _ := api.get_json("/users/by-external/"+fmt.Sprint(external_id)+".json",
		true)
	if j, ok := j.(map[string]interface{}); ok {
		if u, ok := j["user"].(map[string]interface{}); ok {
			return api.parse_user(external_id, u)
		}
	}
	return nil
}

var avatar_size_rexp = regexp.MustCompile("{size}")

func (t *Talk_user) Avatar_url() string {
	return string(avatar_size_rexp.ReplaceAll(t.avatar_url, []byte("120")))
}

//TODO: grab external_id from posters
type Message struct {
	Url             string
	Title           string
	Read            bool
	Last_post       time.Time
	First_post      time.Time
	Reply_count     int
	Poster_avatars  map[string]string
	Last_poster     string
	Original_poster string
}

func (t *Talk_user) Get_messages(limit int) []*Message {
	msgs := make([]*Message, 0)
	usernames := make(map[int]string)
	avatars := make(map[int]string)
	j, _ := t.get_json("/topics/private-messages/"+t.Username+".json", true)
	if j, ok := j.(map[string]interface{}); ok {
		if u, ok := j["users"].([]interface{}); ok {
			for _, v := range u {
				user := v.(map[string]interface{})
				id := int(user["id"].(float64))
				usernames[id] = user["username"].(string)
				avatars[id] = string(avatar_size_rexp.ReplaceAll(
					[]byte(user["avatar_template"].(string)), []byte("120")))
			}
		}
		if tp, ok := j["topic_list"].(map[string]interface{}); ok {
			if tp, ok := tp["topics"].([]interface{}); ok {
				var c int
				for _, v := range tp {
					msg := &Message{Read: true}
					topic := v.(map[string]interface{})
					id := int(topic["id"].(float64))
					slug := topic["slug"].(string)
					msg.Url = t.Path + "/t/" + slug + "/" + fmt.Sprint(id)
					msg.Title = topic["title"].(string)
					msg.Reply_count = int(topic["posts_count"].(float64)) - 1
					msg.First_post, _ = time.ParseInLocation(
						"2006-01-02T15:04:05.999Z",
						topic["created_at"].(string), time.Local)
					msg.Last_post, _ = time.ParseInLocation(
						"2006-01-02T15:04:05.999Z",
						topic["last_posted_at"].(string), time.Local)
					if topic["unseen"].(bool) == true {
						msg.Read = false
					} else if l := topic["highest_post_number"].(float64);
						l != 0 {
						msg.Url += "/" + fmt.Sprint(int(l))
					}
					msg.Last_poster = topic["last_poster_username"].(string)
					msg.Poster_avatars = make(map[string]string)
					for _, p := range topic["posters"].([]interface{}) {
						if p, ok := p.(map[string]interface{}); ok {
							i := int(p["user_id"].(float64))
							msg.Poster_avatars[usernames[i]] = avatars[i]
							if strings.Contains(p["description"].(string),
								"Original Poster") {
								msg.Original_poster = usernames[i]
							}
						}
					}
					msgs = append(msgs, msg)
					if c++; limit == c {
						break
					}
				}
			}
		}
	}
	return msgs
}

func (t *Talk_user) Add_to_group(group string) {
	if _, ok := t.Groups()[group]; !ok {
		log.Println("'", group, "' is not a valid group.")
		return
	}
	form := url.Values{}
	form.Add("user_ids", fmt.Sprint(t.id))
	data := t.put_json("/groups/"+fmt.Sprint(t.Groups()[group])+"/members",
		form)
	j, ok := data.(map[string]interface{})
	if ok {
		if _, ok := j["success"]; ok {
			return
		}
	}
	log.Printf("Talk error on adding %s to group %s: %q\n",
		t.Username, group, j)
}

func (t *Talk_user) Remove_from_group(group string) {
	if _, ok := t.Groups()[group]; !ok {
		log.Println("'", group, "' is not a valid group.")
		return
	}
	form := url.Values{}
	form.Set("user_id", fmt.Sprint(t.id))
	data := t.do_form("DELETE",
		"/groups/"+fmt.Sprint(t.Groups()[group])+"/members", form)
	j, ok := data.(map[string]interface{})
	if ok {
		if _, ok := j["success"]; ok {
			return
		}
	}
	log.Printf("Talk error on removing %s from group %s: %q\n",
		t.Username, group, j)
}

func (t *Talk_user) Activate() {
	t.put_json("/admin/users/"+fmt.Sprint(t.id)+"/activate", url.Values{})
}
