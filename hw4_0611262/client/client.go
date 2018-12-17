package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-stomp/stomp"
)

type response struct {
	Status  int
	Token   string
	Message string
	Friend  []string
	Invite  []string
	Group   []string
	Post    []struct {
		Id      string
		Message string
	}
}

type message struct {
	Source  string `json:"source"`
	Group   string `json:"group,omitempty"`
	Message string `json:"message"`
}

var tokens = make(map[string]string)
var ip = "127.0.0.1:8081"
var stompIp = "127.0.0.1:61613"
var options []func(*stomp.Conn) error = []func(*stomp.Conn) error{
	stomp.ConnOpt.Login("admin", "admin"),
	stomp.ConnOpt.HeartBeatError(1000 * time.Hour),
}
var connected = make(map[string]bool)

func printResp(cmd string, resp response) {
	if resp.Status != 0 {
		fmt.Println(resp.Message)
		return
	}
	switch cmd {
	case "list-invite":
		if len(resp.Invite) > 0 {
			fmt.Println(strings.Join(resp.Invite, "\n"))
		} else {
			fmt.Println("No invitations")
		}
	case "list-friend":
		if len(resp.Friend) > 0 {
			fmt.Println(strings.Join(resp.Friend, "\n"))
		} else {
			fmt.Println("No friends")
		}
	case "receive-post":
		if len(resp.Post) > 0 {
			for _, post := range resp.Post {
				fmt.Println(post.Id + ": " + post.Message)
			}
		} else {
			fmt.Println("No posts")
		}
	case "list-group", "list-joined":
		if len(resp.Group) > 0 {
			for _, group := range resp.Group {
				fmt.Println(group)
			}
		} else {
			fmt.Println("No groups")
		}
	default:
		fmt.Println(resp.Message)
	}
}

func replaceUser(fields []string) (string, string) {
	if len(fields) < 2 {
		return "", strings.Join(fields, " ")
	}
	username := fields[1]
	if fields[0] != "register" && fields[0] != "login" {
		if token, ok := tokens[fields[1]]; ok {
			fields[1] = token
		} else {
			fields[1] = ""
		}
	}
	return username, strings.Join(fields, " ")
}

func dial(cmd string) {
	conn, err := net.Dial("tcp", ip)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return
	}
	username, cmd := replaceUser(fields)
	fmt.Fprint(conn, cmd)
	bytes, _ := ioutil.ReadAll(conn)
	resp := response{}
	json.Unmarshal(bytes, &resp)
	if fields[0] == "login" && resp.Status == 0 {
		tokens[fields[1]] = resp.Token
		subscribed := make(chan bool)
		if _, prs := connected[username]; !prs {
			go recvMessages(fields[1], subscribed)
		}
		<-subscribed
	}
	printResp(fields[0], resp)
}

func recvMessages(username string, subscribed chan bool) {
	stompConn, err := stomp.Dial("tcp", stompIp, options...)
	if err != nil {
		panic(err)
	}
	sub, err := stompConn.Subscribe("/queue/"+username, stomp.AckAuto)
	if err != nil {
		panic(err)
	}
	close(subscribed)
	defer stompConn.Disconnect()
	for {
		resp := <-sub.C
		msg := message{}
		json.Unmarshal(resp.Body, &msg)
		if msg.Group != "" {
			fmt.Printf("<<<%s->GROUP<%s>: %s>>>\n", msg.Source, msg.Group, msg.Message)
		} else {
			fmt.Printf("<<<%s->%s: %s>>>\n", msg.Source, username, msg.Message)
		}

	}
}

func main() {
	if len(os.Args) >= 3 {
		ip = os.Args[1] + ":" + os.Args[2]
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if cmd := scanner.Text(); cmd == "exit" {
			break
		} else {
			dial(cmd)
		}
		time.Sleep(time.Millisecond * 100)
	}
}
