package main

import (
    "encoding/json"
    "flag"
    "io/ioutil"
    "log"
    "github.com/vvanpo/makerspace/site"
)

var Config struct{
    Dir string
    Database struct{
        Conninfo map[string]string
    }
    Beanstream struct{
        Api_key string
        Merchant_id string
    }
    Discourse struct{
        Api_key string
    }
}

func init () {
    var config_filename string
    flag.StringVar(&config_filename, "c", "", "-c [file]")
    flag.Parse()
    config_file, err := ioutil.ReadFile(config_filename)
    if err != nil { log.Fatal("Config file error: ",  err) }
    err = json.Unmarshal(config_file, &Config)
    if err != nil { log.Fatal("Config file error: ",  err) }
}

func main () {
    db := Database(Config.Database.Conninfo)
    site.Serve(":1080", Config.Dir + "/site", db)
}
