package main

import (
	"fmt"
	"geerpc"
	"log"
	"net"
	"sync"
	"time"
)

// 在 startServer 中使用了信道addr，确保服务端端口监听成功，客户端再发起请求
// 客户端首先发送Option进行协议交换，接下来发送消息头h:=&codec.Header{}，和消息体geerpc req ${h.Seq}
// 最后解析服务端的响应reply，并打印出来
func main()  {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)
	client, _ := geerpc.Dial("tcp", <-addr)
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)
	// send request & receive response
	var wg sync.WaitGroup
	for i:= 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("geerpc req %d", i)
			var reply string
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}

func startServer(addr chan string)  {
	// pick a free port
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	geerpc.Accept(l)
}
