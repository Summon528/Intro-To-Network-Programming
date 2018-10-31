package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

var db = ConnectDB()

func handleConnection(conn net.Conn) {
	buf := make([]byte, 1048576)
	bufLen, _ := conn.Read(buf)
	response := HandleCmd(string(buf[:bufLen]))
	json, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(json))
	fmt.Fprint(conn, string(json))
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
