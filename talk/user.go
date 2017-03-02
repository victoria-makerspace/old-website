package talk

import (
	"log"
	"net/url"
	"regexp"
	"fmt"
	"time"
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
	Active		   bool
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
	if active, ok := u["active"].(bool); ok {
		t.Active = active
	}
	t.Username = u["username"].(string)
	t.avatar_url = []byte(u["avatar_template"].(string))
	if card, ok := u["card_background"].(string); ok {
		t.Card_bg_url = api.Base_url + card
	}
	if profile, ok := u["profile_background"].(string); ok {
		t.Profile_bg_url = api.Base_url + profile
	}
	return t
}

func (api *Talk_api) Get_user(id int) *Talk_user {
	t := &Talk_user{external_id: id, Talk_api: api}
	j := api.get_json("/users/by-external/"+fmt.Sprint(id)+".json", api.admin)
	if j, ok := j.(map[string]interface{}); ok {
		if u, ok := j["user"].(map[string]interface{}); ok {
			t = api.parse_user(id, u)
		}
	}
	if t == nil {
		return nil
	}
	j = api.get_json("/admin/users/"+fmt.Sprint(t.id)+".json", api.admin)
	if j, ok := j.(map[string]interface{}); ok {
		if a, ok := j["active"].(bool); ok {
			t.Active = a
		}
	}
	return t
}

func (t *Talk_user) Send_activation_email() {
	values := url.Values{}
	values.Set("username", t.Username)
	t.post_json("/users/action/send_activation_email", values)
}

var avatar_size_rexp = regexp.MustCompile("{size}")

func (t *Talk_user) Avatar_url(size int) string {
	return t.Base_url + string(avatar_size_rexp.ReplaceAll(t.avatar_url,
		[]byte(fmt.Sprint(size))))
}

type Message struct {
	Url string
	Title string
	Read bool
	Last_post time.Time
	First_post time.Time
	Post_count int
	Poster_avatars map[string]string
	Last_poster string
}

func (t *Talk_user) Get_messages(limit int) []*Message {
	msgs := make([]*Message, 0)
	usernames := make(map[int]string)
	avatars := make(map[int]string)
	j := t.get_json("/topics/private-messages/" + t.Username + ".json")
	if j, ok := j.(map[string]interface{}); ok {
		if u, ok := j["users"].([]interface{}); ok {
			for _, v := range u {
				user := v.(map[string]interface{})
				id := int(user["id"].(float64))
				usernames[id] = user["username"].(string)
				avatars[id] = user["avatar_template"].(string)
			}
		}
		if tp, ok := j["topic_list"].(map[string]interface{}); ok {
			if tp, ok := tp["topics"].([]interface{}); ok {
				var c int
				for _, v := range tp {
					msg := &Message{}
					topic := v.(map[string]interface{})
					id := int(topic["id"].(float64))
					slug := topic["slug"].(string)
					msg.Url = t.Url() + "/t/" + slug + "/" + fmt.Sprint(id)
					msg.Title = topic["title"].(string)
					msg.Post_count = int(topic["posts_count"].(float64))
					msg.First_post, _ = time.ParseInLocation(
						"2006-01-02T15:04:05.999Z",
						topic["created_at"].(string), time.Local)
					msg.Last_post, _ = time.ParseInLocation(
						"2006-01-02T15:04:05.999Z",
						topic["last_posted_at"].(string), time.Local)
					if topic["unread"].(float64) == 0 {
						msg.Read = true
					}
					if l := topic["last_read_post_number"].(float64); l != 0 {
						url += "/" + fmt.Sprint(int(l))
					}
					msg.Last_poster = topic["last_poster_username"].(string)
					msgs = append(msgs, msg)
					log.Println(msg)
					if c++; limit == c {
						break
					}
				}
			}
		}
	}
	return msgs
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
