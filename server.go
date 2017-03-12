package main

import (
	"encoding/json"
	"flag"
	"github.com/vvanpo/makerspace/billing"
	"github.com/vvanpo/makerspace/member"
	"github.com/vvanpo/makerspace/site"
	"github.com/vvanpo/makerspace/talk"
	"io/ioutil"
	"log"
	"os"
	"path"
)

var config struct {
	Site       map[string]interface{}
	Members    map[string]interface{}
	Database   map[string]string
	Beanstream map[string]string
	Talk       map[string]string
	Smtp       map[string]string
}

func init() {
	var config_filepath string
	flag.StringVar(&config_filepath, "c", "", "-c [file]")
	flag.Parse()
	if config_filepath == "" {
		config_filepath = path.Dir(os.Args[0]) + "/config.json"
	}
	config_file, err := ioutil.ReadFile(config_filepath)
	if err != nil {
		log.Fatal("Config file error: ", err)
	}
	err = json.Unmarshal(config_file, &config)
	if err != nil {
		log.Fatal("Config file error: ", err)
	}
}

func main() {
	db := Database(config.Database)
	bs := config.Beanstream
	talk := talk.New_talk_api(config.Talk)
	b := billing.Billing_new(bs["merchant-id"], bs["payments-api-key"],
		bs["profiles-api-key"], bs["reports-api-key"], db)
	members := &member.Members{config.Members, db, talk, b}
	site.Serve(config.Site, talk, members, db)
}
