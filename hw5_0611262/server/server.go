package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-stomp/stomp"
)

var db = ConnectDB()
var tokens = make(map[string]string)
var options = []func(*stomp.Conn) error{
	stomp.ConnOpt.Login("user", AWSPassword),
	stomp.ConnOpt.HeartBeatError(1000 * time.Hour),
}

func handleConnection(conn net.Conn) {
	buf := make([]byte, 1048576)
	bufLen, _ := conn.Read(buf)
	response := HandleCmd(string(buf[:bufLen]))
	fmt.Println(response)
	fmt.Fprint(conn, response)
	conn.Close()
}

func main() {
	ip := "0.0.0.0:8888"
	if len(os.Args) >= 3 {
		ip = os.Args[1] + ":" + os.Args[2]
	}
	defer db.Close()
	ln, _ := net.Listen("tcp", ip)
	resp, err := http.Get("http://instance-data/latest/meta-data/instance-id")
	if err == nil {
		body, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		db.FirstOrCreate(&Instance{}, Instance{ID: string(body)})
	}
	println("Start Listening")
	for {
		conn, _ := ln.Accept()
		go handleConnection(conn)
	}
}
