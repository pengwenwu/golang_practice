package main

import (
	"flag"
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
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://127.0.0.1:9999"
	addrMap := map[int]string{
		8001: "http://127.0.0.1:8001",
		8002: "http://127.0.0.1:8002",
		8003: "http://127.0.0.1:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup()
	if api {
		go startAPIServer(apiAddr, gee)
	}
	startCacheSever(addrMap[port], []string(addrs), gee)
}

func createGroup() *geekcache.Group {
	return geekcache.NewGroup("scores", 2<<20, geekcache.GetterFunc(func(key string) ([]byte, error) {
		log.Println("[SlowDB search key]", key)
		if v, ok := db[key]; ok {
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s not exist", key)
	}))
}

// 启动缓存服务器：创建HTTPPool，添加节点信息，注册到gee中，启动HTTP服务，用户无感知
func startCacheSever(addr string, adds []string, gee *geekcache.Group) {
	peers := geekcache.NewHTTPPool(addr)
	peers.Set(adds...)
	gee.RegisterPeers(peers)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// 启动一个API服务（端口9999），与用户进行交互，用户感知
func startAPIServer(apiAddr string, gee *geekcache.Group) {
	http.Handle("/api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		view, err := gee.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(view.ByteSlice())
	}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}
