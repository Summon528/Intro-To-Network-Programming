package main

import (
	"encoding/json"
	"strings"

	"github.com/go-stomp/stomp"
	"github.com/google/uuid"
)

type responsePost struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func HandleCmd(cmd string) string {
	cmdSplit := strings.SplitN(cmd, " ", 2)
	cmdType := cmdSplit[0]
	cmdMsg := ""
	if len(cmdSplit) >= 2 {
		cmdMsg = cmdSplit[1]
	}
	var result *map[string]interface{}
	switch cmdType {
	case "register":
		result = handleRegister(cmdMsg)
	case "login":
		result = handleLogin(cmdMsg)
	case "delete":
		result = handleDelete(cmdMsg)
	case "logout":
		result = handleLogout(cmdMsg)
	case "invite":
		result = handleInvite(cmdMsg)
	case "list-invite":
		result = handleListInvite(cmdMsg)
	case "accept-invite":
		result = handleAcceptInvite(cmdMsg)
	case "list-friend":
		result = handleListFriend(cmdMsg)
	case "post":
		result = handlePost(cmdMsg)
	case "receive-post":
		result = handleRecivePost(cmdMsg)
	case "send":
		result = handleSend(cmdMsg)
	case "create-group":
		result = handleCreateGroup(cmdMsg)
	case "list-group":
		result = handleListGroup(cmdMsg)
	case "list-joined":
		result = handleListJoined(cmdMsg)
	case "join-group":
		result = handleJoinGroup(cmdMsg)
	case "send-group":
		result = handleSendGroup(cmdMsg)
	default:
		result = &map[string]interface{}{"status": 1, "message": "Unknown command " + cmdType}
	}
	resJSON, _ := json.Marshal(result)
	return string(resJSON)
}

func stupidToken(cmdMsg string) (*map[string]interface{}, bool) {
	fields := strings.Fields(cmdMsg)
	if len(fields) == 0 || db.Where("token = ?", fields[0]).First(&User{}).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": "Not login yet"}, false
	}
	return &map[string]interface{}{}, true
}

func handleRegister(cmdMsg string) *map[string]interface{} {
	fields := strings.Fields(cmdMsg)
	if len(fields) != 2 {
		return &map[string]interface{}{"status": 1, "message": "Usage: register <id> <password>"}
	}
	username, password := fields[0], fields[1]
	if !db.Where("username = ?", username).First(&User{}).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": username + " is already used"}
	}
	db.Create(&User{Username: username, Password: password})
	return &map[string]interface{}{"status": 0, "message": "Success!"}
}

func handleLogin(cmdMsg string) *map[string]interface{} {
	fields := strings.Fields(cmdMsg)
	if len(fields) != 2 {
		return &map[string]interface{}{"status": 1, "message": "Usage: login <id> <password>"}
	}
	var user User
	username, password := fields[0], fields[1]
	if db.Where("username = ? AND password = ?", username, password).First(&user).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": "No such user or password error"}
	}
	if user.Token == "" {
		user.Token = uuid.New().String()
		db.Save(&user)
	}
	return &map[string]interface{}{"status": 0, "token": user.Token, "message": "Success!"}
}

func handleDelete(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return &map[string]interface{}{"status": 1, "message": "Usage: delete <user>"}
	}
	var user User
	token := fields[0]
	db.Where("token = ?", token).First(&user)
	db.Delete(Post{}, "owner_id = ?", user.ID)
	db.Exec("DELETE FROM friendships WHERE user_id = ? OR friend_id = ?", user.ID, user.ID)
	db.Exec("DELETE FROM invites WHERE user_id = ? OR friend_id = ?", user.ID, user.ID)
	db.Exec("DELETE FROM group_members WHERE user_id = ?", user.ID)
	db.Delete(&user)
	return &map[string]interface{}{"status": 0, "message": "Success!"}
}

func handleLogout(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return &map[string]interface{}{"status": 1, "message": "Usage: logout <user>"}
	}
	var user User
	token := fields[0]
	db.Where("token = ?", token).First(&user)
	user.Token = ""
	db.Save(&user)
	return &map[string]interface{}{"status": 0, "message": "Bye!"}
}

func handleInvite(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 2 {
		return &map[string]interface{}{"status": 1, "message": "Usage: invite <user> <id>"}
	}
	var user User
	token, friendName := fields[0], fields[1]
	db.Where("token = ?", token).First(&user)
	if friendName == user.Username {
		return &map[string]interface{}{"status": 1, "message": "You cannot invite yourself"}
	}
	var friend User
	if db.Where("username = ?", friendName).Preload("Invites", "friend_id = ?", user.ID).First(&friend).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": friendName + " does not exist"}
	}
	if len(friend.Invites) > 0 {
		return &map[string]interface{}{"status": 1, "message": "Already invited"}
	}
	var invites []User
	db.Model(&user).Where("friend_id = ?", friend.ID).Related(&invites, "Invites")
	if len(invites) > 0 {
		return &map[string]interface{}{"status": 1, "message": friendName + " has invited you"}
	}
	var friends []User
	db.Model(&user).Where("friend_id = ?", friend.ID).Related(&friends, "Friends")
	if len(friends) > 0 {
		return &map[string]interface{}{"status": 1, "message": friendName + " is already your friend"}
	}
	db.Model(&friend).Association("Invites").Append(&user)
	return &map[string]interface{}{"status": 0, "message": "Success!"}
}

func handleListInvite(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return &map[string]interface{}{"status": 1, "message": "Usage: list-invite <user>"}
	}
	var user User
	token := fields[0]
	db.Where("token = ?", token).Preload("Invites").First(&user)
	invites := make([]string, 0)
	for _, invite := range user.Invites {
		invites = append(invites, invite.Username)
	}
	return &map[string]interface{}{"status": 0, "invite": invites}
}

func handleAcceptInvite(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 2 {
		return &map[string]interface{}{"status": 1, "message": "Usage: accept-invite <user> <id>"}
	}
	var user User
	token, friendName := fields[0], fields[1]
	db.Where("token = ?", token).Preload("Invites", "username = ?", friendName).First(&user)
	if len(user.Invites) == 0 {
		return &map[string]interface{}{"status": 1, "message": friendName + " did not invite you"}
	}
	var friend = user.Invites[0]
	db.Model(&user).Association("Invites").Delete(friend)
	db.Model(&user).Association("Friends").Append(friend)
	db.Model(&friend).Association("Friends").Append(user)
	return &map[string]interface{}{"status": 0, "message": "Success!"}
}

func handleListFriend(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return &map[string]interface{}{"status": 1, "message": "Usage: list-friend <user>"}
	}
	var user User
	token := fields[0]
	db.Where("token = ?", token).Preload("Friends").First(&user)
	friends := make([]string, 0)
	for _, friend := range user.Friends {
		friends = append(friends, friend.Username)
	}
	return &map[string]interface{}{"status": 0, "friend": friends}
}

func handlePost(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.SplitN(cmdMsg, " ", 2)
	if len(fields) != 2 {
		return &map[string]interface{}{"status": 1, "message": "Usage: post <user> <message>"}
	}
	var user User
	token, message := fields[0], fields[1]
	db.Where("token = ?", token).First(&user)
	db.Create(&Post{OwnerID: user.ID, Message: message})
	return &map[string]interface{}{"status": 0, "message": "Success!"}
}

func handleRecivePost(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return &map[string]interface{}{"status": 1, "message": "Usage: receive-post <user>"}
	}
	var user User
	token := fields[0]
	db.Where("token = ?", token).Preload("Friends").First(&user)
	var friends []uint
	for _, friend := range user.Friends {
		friends = append(friends, friend.ID)
	}
	var posts []Post
	db.Where("owner_id in (?)", friends).Preload("Owner").Find(&posts)
	responsePosts := make([]responsePost, 0)
	for _, post := range posts {
		responsePosts = append(responsePosts, responsePost{ID: post.Owner.Username, Message: post.Message})
	}
	return &map[string]interface{}{"status": 0, "post": responsePosts}
}

type message struct {
	Source  string `json:"source"`
	Group   string `json:"group,omitempty"`
	Message string `json:"message"`
}

func handleSend(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) < 3 {
		return &map[string]interface{}{"status": 1, "message": "Usage: send <user> <friend> <message>"}
	}
	var friend User
	friendName := fields[1]
	if db.Where("username = ?", friendName).Preload("Friends").First(&friend).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": "No such user exist"}
	}
	token := fields[0]
	var user User
	db.Where("token = ?", token).First(&user)
	flag := false
	for _, i := range friend.Friends {
		if i.Username == user.Username {
			flag = true
			break
		}
	}
	if !flag {
		return &map[string]interface{}{"status": 1, "message": friendName + " is not your friend"}
	}
	if friend.Token == "" {
		return &map[string]interface{}{"status": 1, "message": friendName + " is not online"}
	}
	msg, _ := json.Marshal(message{Source: user.Username, Message: strings.Join(fields[2:], " ")})
	stompConn, err := stomp.Dial("tcp", stompIp, options...)
	if err != nil {
		panic(err)
	}
	defer stompConn.Disconnect()
	stompConn.Send("/queue/"+friendName, "text/plain", []byte(msg))
	return &map[string]interface{}{"status": 0, "message": "Success!"}
}

func handleCreateGroup(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	token := fields[0]
	var user User
	db.Where("token = ?", token).First(&user)
	if len(fields) != 2 {
		return &map[string]interface{}{"status": 1, "message": "Usage: create-group <user> <group>"}
	}
	groupName := fields[1]
	if !db.Where("groupname = ?", groupName).First(&Group{}).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": groupName + " already exist"}
	}
	db.Create(&Group{Groupname: groupName, Members: []*User{&user}})
	return &map[string]interface{}{"status": 0, "message": "Success!"}
}

func handleListGroup(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return &map[string]interface{}{"status": 1, "message": "Usage: list-group <user>"}
	}
	var groups []Group
	db.Find(&groups)
	resp := make([]string, 0)
	for _, group := range groups {
		resp = append(resp, group.Groupname)
	}
	return &map[string]interface{}{"status": 0, "group": resp}
}

func handleListJoined(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return &map[string]interface{}{"status": 1, "message": "Usage: list-joined <user>"}
	}
	var user User
	token := fields[0]
	db.Where("token = ?", token).Preload("Groups").First(&user)
	resp := make([]string, 0)
	for _, group := range user.Groups {
		resp = append(resp, group.Groupname)
	}
	return &map[string]interface{}{"status": 0, "group": resp}
}

func handleJoinGroup(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 2 {
		return &map[string]interface{}{"status": 1, "message": "Usage: join-group <user> <group>"}
	}
	groupName := fields[1]
	var group Group
	if db.Where("groupname = ?", groupName).First(&group).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": groupName + " does not exist"}
	}
	var user User
	token := fields[0]
	db.Where("token = ?", token).Preload("Groups").First(&user)
	for _, i := range user.Groups {
		if i.Groupname == groupName {
			return &map[string]interface{}{"status": 1, "message": "Already a member of " + groupName}
		}
	}
	db.Model(&group).Association("members").Append(&user)
	return &map[string]interface{}{"status": 0, "message": "Success!"}
}

func handleSendGroup(cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) < 3 {
		return &map[string]interface{}{"status": 1, "message": "Usage: send-group <user> <group> <message>"}
	}
	token := fields[0]
	groupName := fields[1]
	var group Group
	var user User
	if db.Where("groupname = ?", groupName).Preload("Members").First(&group).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": "No such group exist"}
	}
	db.Where("token = ?", token).First(&user)
	flag := false
	for _, memeber := range group.Members {
		if memeber.ID == user.ID {
			flag = true
			break
		}
	}
	if !flag {
		return &map[string]interface{}{"status": 1, "message": "You are not the member of " + groupName}
	}
	msg, _ := json.Marshal(
		message{
			Source:  user.Username,
			Message: strings.Join(fields[2:], " "),
			Group:   groupName,
		})
	for _, member := range group.Members {
		stompConn, err := stomp.Dial("tcp", stompIp, options...)
		if err != nil {
			panic(err)
		}
		defer stompConn.Disconnect()
		if member.Token != "" {
			stompConn.Send("/queue/"+member.Username, "text/plain", []byte(msg))
		}
	}
	return &map[string]interface{}{"status": 0, "message": "Success!"}
}
