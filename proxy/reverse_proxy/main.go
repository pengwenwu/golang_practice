package main

import (
	"log"
	"net/http"
)

var cmd Cmd
var srv http.Server

func main() {
	cmd = parseCmd()
	StartServer(cmd.bind, cmd.remote)
}

func StartServer(bind string, remote string) {
	log.Printf("Listening on %s, forwarding to %s", bind, remote)
	h := &handle{reverseProxy: remote}
	srv.Addr = bind
	srv.Handler = h
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalln("ListenAdnServ: ", err)
	}
}

func StopServer() {
	if err := srv.Shutdown(nil); err != nil {
		log.Println(err)
	}
}
