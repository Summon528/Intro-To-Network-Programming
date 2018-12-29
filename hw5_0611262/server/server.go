package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/go-stomp/stomp"
)

var db = ConnectDB()
var tokens = make(map[string]string)
var ip = "127.0.0.1:8081"
var stompIp = "127.0.0.1:61613"
var options []func(*stomp.Conn) error = []func(*stomp.Conn) error{
	stomp.ConnOpt.Login("admin", "admin"),
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
	ip := "127.0.0.1:8081"
	if len(os.Args) >= 3 {
		ip = os.Args[1] + ":" + os.Args[2]
	}
	defer db.Close()
	ln, _ := net.Listen("tcp", ip)
	for {
		conn, _ := ln.Accept()
		go handleConnection(conn)
	}
}
