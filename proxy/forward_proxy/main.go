package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strings"
)

func main() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Panic(err)
	}

	// 死循环，每当遇到连接的时候，调用handle
	for {
		client, err := l.Accept()
		if err != nil {
			log.Panic(err)
		}

		go handle(client)
	}
}

func handle(client net.Conn)  {
	if client == nil {
		return
	}
	defer client.Close()

	log.Printf("remote addr: %v\n", client.RemoteAddr())

	// 将获取的数据放入缓冲区：
	// 用来存放客户端数据的缓冲区
	var b [1024]byte
	// 从客户端获取数据
	n, err := client.Read(b[:])
	if err != nil {
		log.Println(err)
		return
	}

	// 从缓冲区读取HTTP请求方法，URL等信息：
	var method, URL, address string
	// 从客户端数据读入method，url
	fmt.Sscanf(string(b[:bytes.IndexByte(b[:], '\n')]), "%s%s", &method, &URL)
	hostPortURL, err := url.Parse(URL)
	if err != nil {
		log.Println(err)
		return
	}

	// http协议和https协议获取地址的方式不同，分别处理：
	// 如果方法是CONNECT，则为https协议
	if method == "CONNECT" {
		address = hostPortURL.Scheme + ":" + hostPortURL.Opaque
	} else {
		// 否则为http协议
		address = hostPortURL.Host
		// 如果host不带端口，则默认为80
		if strings.Index(hostPortURL.Host, ":") == -1 {
			address = hostPortURL.Host + ":80"
		}
	}

	// 用获取到的地址向服务端发起请求。如果是http协议，将客户端的请求直接转发给服务端；如果是https协议，发送http响应
	// 获得了请求的host和port，向服务端发起tcp连接
	server, err := net.Dial("tcp", address)
	if err != nil {
		log.Println(err)
		return
	}
	// 如果使用https协议，需要先向客户端表示连接建立完毕
	if method == "CONNECT" {
		fmt.Fprint(client, "HTTP/1.1 200 Connection established\r\n\r\n")
	} else {
		// 如果使用http协议，需要将从客户端得到的http请求转发给服务端
		server.Write(b[:n])
	}

	// 最后，将所有客户端的请求转发至服务端，将所有服务端的响应转发给客户端.
	// io.Copy 为阻塞函数；文件描述符不关闭就不停止
	go io.Copy(server, client)
	io.Copy(client, server)
}
