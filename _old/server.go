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
	Site     site.Config
	Members  member.Config
	Database map[string]string
	Talk     talk.Api
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
	if config.Site.Talk_proxy != "" {
		config.Talk.Url = config.Site.Talk_proxy + config.Talk.Path
	} else {
		config.Talk.Url = config.Site.Url() + config.Talk.Path
	}
}

func main() {
	db := Database(config.Database)
	talk := &config.Talk
	members := member.New(config.Members, db, talk)
	site.Serve(config.Site, members, db)
}
