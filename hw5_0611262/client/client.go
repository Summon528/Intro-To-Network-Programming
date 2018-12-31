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
	Host    string
	Friend  []string
	Invite  []string
	Group   []string
	Post    []struct {
		ID      string
		Message string
	}
}

type message struct {
	Source  string `json:"source"`
	Group   string `json:"group,omitempty"`
	Message string `json:"message"`
}

var tokens = make(map[string]string)
var options = []func(*stomp.Conn) error{
	stomp.ConnOpt.Login("user", AWSPassword),
	stomp.ConnOpt.HeartBeatError(1000 * time.Hour),
}
var connected = make(map[string]bool)
var userServerHost = make(map[string]string)

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
				fmt.Println(post.ID + ": " + post.Message)
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
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return
	}
	destIP := ""
	if fields[0] == "register" ||
		fields[0] == "login" ||
		fields[0] == "delete" ||
		fields[0] == "logout" ||
		len(fields) <= 1 {
		destIP = LoginHost
	} else {
		if val, ok := userServerHost[fields[1]]; ok {
			destIP = val + ":8888"
		} else {
			destIP = LoginHost
		}
	}
	conn, err := net.Dial("tcp", destIP)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	username, cmd := replaceUser(fields)
	fmt.Fprint(conn, cmd)
	bytes, _ := ioutil.ReadAll(conn)
	resp := response{}
	json.Unmarshal(bytes, &resp)
	if fields[0] == "login" && resp.Status == 0 {
		tokens[fields[1]] = resp.Token
		userServerHost[fields[1]] = resp.Host
		subscribed := make(chan bool)
		if _, prs := connected[username]; !prs {
			go recvMessages(fields[1], subscribed)
		}
		<-subscribed
	}
	if (fields[0] == "logout" || fields[0] == "delete") && resp.Status == 0 {
		delete(userServerHost, fields[1])
	}
	printResp(fields[0], resp)
}

func recvMessages(username string, subscribed chan bool) {
	stompConn, err := stomp.Dial("tcp", MQHost, options...)
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
		LoginHost = os.Args[1] + ":" + os.Args[2]
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
