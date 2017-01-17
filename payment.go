package main

import (
    "io/ioutil"
    "log"
    "os"
    beanstream "github.com/Beanstream/beanstream-go"
    "github.com/Beanstream/beanstream-go/paymentMethods"
)

config := beanstream.DefaultConfig()

func init() {
    id, err := ioutil.ReadFile(os.Getenv("MAKERSPACE_DIR") + "/Keys/beanstream-merchant-id")
    if err != nil { log.Fatal(err) }
    key, err := ioutil.ReadFile(os.Getenv("MAKERSPACE_DIR") + "/Keys/beanstream-api-key")
    if err != nil { log.Fatal(err) }
    config.PaymentsApiKey= string(key)
}
