package main

import (
    "encoding/json"
    "flag"
    "io/ioutil"
    "log"
    "os"
    "path"
    "github.com/vvanpo/makerspace/site"
)

var Config struct {
    Domain string
    Port int
    Dir string
    Database struct {
        Conninfo map[string]string
    }
    Beanstream struct {
        Api_key string
        Merchant_id string
    }
    Discourse struct {
        Api_key string
    }
}

func init () {
    var config_filepath string
    flag.StringVar(&config_filepath, "c", "", "-c [file]")
    flag.Parse()
    if config_filepath == "" {
        config_filepath = path.Dir(os.Args[0]) + "/config.json"
    }
    config_file, err := ioutil.ReadFile(config_filepath)
    if err != nil { log.Fatal("Config file error: ",  err) }
    err = json.Unmarshal(config_file, &Config)
    if err != nil { log.Fatal("Config file error: ",  err) }
}

func main () {
    db := Database(Config.Database.Conninfo)
    config := site.Config{
        Config.Domain,
        Config.Port,
        Config.Dir + "/site/templates/",
        Config.Dir + "/site/static/",
        Config.Dir + "/database/data/",
        Config.Beanstream.Merchant_id,
        Config.Beanstream.Api_key,
    }
    site.Serve(config, db)
}
