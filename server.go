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
    var site site.Http_server
    site.Addr = ":1080"
    site.Dir = Config.Dir + "/site"
    site.Serve()
}
