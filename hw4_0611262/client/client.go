package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
)

type response struct {
	Status  int
	Token   string
	Message string
	Friend  []string
	Invite  []string
	Post    []struct {
		Id      string
		Message string
	}
}

var tokens = make(map[string]string)
var ip = os.Args[1]
var port = os.Args[2]

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
	default:
		fmt.Println(resp.Message)
	}
}

func replaceUser(fields []string) string {
	if len(fields) < 2 {
		return strings.Join(fields, " ")
	}
	if fields[0] != "register" && fields[0] != "login" {
		if token, ok := tokens[fields[1]]; ok {
			fields[1] = token
		} else {
			fields[1] = ""
		}
	}
	return strings.Join(fields, " ")
}

func dial(cmd string) {
	conn, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return
	}
	cmd = replaceUser(fields)
	fmt.Fprint(conn, cmd)
	bytes, _ := ioutil.ReadAll(conn)
	resp := response{}
	json.Unmarshal(bytes, &resp)
	if fields[0] == "login" && resp.Status == 0 {
		tokens[fields[1]] = resp.Token
	}
	printResp(fields[0], resp)
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if cmd := scanner.Text(); cmd == "exit" {
			break
		} else {
			dial(cmd)
		}
	}
}
