package main

import (
	"encoding/json"
	"flag"
	"github.com/vvanpo/makerspace/billing"
	"github.com/vvanpo/makerspace/site"
	"io/ioutil"
	"log"
	"os"
	"path"
)

var config struct {
	Domain     string
	Port       int
	Dir        string
	Database   map[string]string
	Beanstream map[string]string
	Discourse map[string]string
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
	b := billing.Billing_new(bs["merchant-id"], bs["payments-api-key"], bs["profiles-api-key"], bs["reports-api-key"], db)
	site.Serve(site.Config{
		config.Domain,
		config.Port,
		config.Dir + "/site/templates/",
		config.Dir + "/site/static/",
		config.Dir + "/database/data/",
		config.Discourse}, db, b)
}
