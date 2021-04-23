package main

import (
	"fmt"
	"geekcache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	geekcache.NewGroup("scores", 2<<10, geekcache.GetterFunc(func(key string) ([]byte, error) {
		log.Println("[SlowDB] search key", key)
		if v, ok := db[key]; ok {
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s not exist", key)
	}))

	addr := "localhost:9999"
	peers := geekcache.NewHTTPPool(addr)
	log.Println("geekcache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}
