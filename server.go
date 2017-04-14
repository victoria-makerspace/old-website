package main

import (
	"encoding/json"
	"flag"
	"github.com/vvanpo/makerspace/member"
	"github.com/vvanpo/makerspace/site"
	"github.com/vvanpo/makerspace/talk"
	"io/ioutil"
	"log"
	"os"
)

var config struct {
	Site       map[string]interface{}
	Members    member.Config
	Database   map[string]string
	Talk       map[string]string
	Smtp       map[string]string
}

func init() {
	var config_filepath string
	flag.StringVar(&config_filepath, "c", "", "-c [file]")
	flag.Parse()
	if config_filepath == "" {
		gopath := os.Getenv("GOPATH")
		config_filepath = gopath + "/src/github.com/vvanpo/makerspace/config.json"
	}
	config_file, err := ioutil.ReadFile(config_filepath)
	if err != nil {
		log.Fatal("Config file error: ", err)
	}
	err = json.Unmarshal(config_file, &config)
	if err != nil {
		log.Fatal("Config file error: ", err)
	}
	tls := config.Site["tls"].(bool)
	url := config.Site["domain"].(string)
	if tls {
		url = "https://" + url
	} else {
		url = "http://" + url
	}
	if proxy, ok := config.Site["talk-proxy"].(string); ok {
		config.Talk["url"] = proxy + config.Talk["path"]
	} else {
		config.Talk["url"] = url + config.Talk["path"]
	}
}

func main() {
	db := Database(config.Database)
	talk := talk.New_talk_api(config.Talk)
	members := member.New(config.Members, db, talk)
	site.Serve(config.Site, talk, members, db)
}
