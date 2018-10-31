package main

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
)

type response struct {
	Status  uint
	Token   string         `json:"token,omitempty"`
	Message string         `json:"message,omitempty"`
	Invite  []string       `json:"invite,omitempty"`
	Friend  []string       `json:"friend,omitempty"`
	Post    []responsePost `json:"post,omitempty"`
}
type responsePost struct {
	ID      string
	Message string
}

func HandleCmd(cmd string) response {
	cmdSplit := strings.SplitN(cmd, " ", 2)
	cmdType := cmdSplit[0]
	cmdMsg := ""
	if len(cmdSplit) >= 2 {
		cmdMsg = cmdSplit[1]
	}
	switch cmdType {
	case "register":
		return handleRegister(db, cmdMsg)
	case "login":
		return handleLogin(db, cmdMsg)
	case "delete":
		return handleDelete(db, cmdMsg)
	case "logout":
		return handleLogout(db, cmdMsg)
	case "invite":
		return handleInvite(db, cmdMsg)
	case "list-invite":
		return handleListInvite(db, cmdMsg)
	case "accept-invite":
		return handleAcceptInvite(db, cmdMsg)
	case "list-friend":
		return handleListFriend(db, cmdMsg)
	case "post":
		return handlePost(db, cmdMsg)
	case "receive-post":
		return handleRecivePost(db, cmdMsg)
	default:
		return response{Status: 1, Message: "Unknown command " + cmdType}
	}
}

func stupidToken(cmdMsg string) (response, bool) {
	fields := strings.Fields(cmdMsg)
	if len(fields) == 0 || db.Where("token = ?", fields[0]).First(&User{}).RecordNotFound() {
		return response{Status: 1, Message: "Not login yet"}, false
	}
	return response{}, true
}

func handleRegister(db *gorm.DB, cmdMsg string) response {
	fields := strings.Fields(cmdMsg)
	if len(fields) != 2 {
		return response{Status: 1, Message: "Usage: register <id> <password>"}
	}
	username, password := fields[0], fields[1]
	if !db.Where("username = ?", username).First(&User{}).RecordNotFound() {
		return response{Status: 1, Message: username + " is already used"}
	}
	db.Create(&User{Username: username, Password: password})
	return response{Status: 0, Message: "Success!"}
}

func handleLogin(db *gorm.DB, cmdMsg string) response {
	fields := strings.Fields(cmdMsg)
	if len(fields) != 2 {
		return response{Status: 1, Message: "Usage: login <id> <password>"}
	}
	var user User
	username, password := fields[0], fields[1]
	if db.Where("username = ? AND password = ?", username, password).First(&user).RecordNotFound() {
		return response{Status: 1, Message: "No such user or password error"}
	}
	if user.Token == "" {
		user.Token = uuid.New().String()
		db.Save(&user)
	}
	return response{Status: 0, Token: user.Token, Message: "Success!"}
}

func handleDelete(db *gorm.DB, cmdMsg string) response {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return response{Status: 1, Message: "Usage: delete <user>"}
	}
	var user User
	token := fields[0]
	if db.Where("token = ?", token).First(&user).RecordNotFound() {
		return response{Status: 1, Message: "Not login yet"}
	}
	db.Delete(Post{}, "owner_id = ?", user.ID)
	db.Exec("DELETE FROM friendships WHERE user_id = ? OR friend_id = ?", user.ID, user.ID)
	db.Exec("DELETE FROM invites WHERE user_id = ? OR friend_id = ?", user.ID, user.ID)
	db.Delete(&user)
	return response{Status: 0, Message: "Success!"}
}

func handleLogout(db *gorm.DB, cmdMsg string) response {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return response{Status: 1, Message: "Usage: logout <user>"}
	}
	var user User
	token := fields[0]
	if db.Where("token = ?", token).First(&user).RecordNotFound() {
		return response{Status: 1, Message: "Not login yet"}
	}
	user.Token = ""
	db.Save(&user)
	return response{Status: 0, Message: "Bye!"}
}

func handleInvite(db *gorm.DB, cmdMsg string) response {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 2 {
		return response{Status: 1, Message: "Usage: invite <user> <id>"}
	}
	var user User
	token, friendName := fields[0], fields[1]
	if db.Where("token = ?", token).First(&user).RecordNotFound() {
		return response{Status: 1, Message: "Not login yet"}
	}
	if friendName == user.Username {
		return response{Status: 1, Message: "You cannot invite yourself"}
	}
	var friend User
	if db.Where("username = ?", friendName).Preload("Invites", "friend_id = ?", user.ID).First(&friend).RecordNotFound() {
		return response{Status: 1, Message: friendName + " does not exist"}
	}
	fmt.Print(friend.Invites)
	if len(friend.Invites) > 0 {
		return response{Status: 1, Message: "Already invited"}
	}
	var invites []User
	db.Model(&user).Where("friend_id = ?", friend.ID).Related(&invites, "Invites")
	if len(invites) > 0 {
		return response{Status: 1, Message: friendName + " has invited you"}
	}
	var friends []User
	db.Model(&user).Where("friend_id = ?", friend.ID).Related(&friends, "Friends")
	if len(friends) > 0 {
		return response{Status: 1, Message: friendName + " is already your friend"}
	}
	db.Model(&friend).Association("Invites").Append(&user)
	return response{Status: 0, Message: "Success!"}
}

func handleListInvite(db *gorm.DB, cmdMsg string) response {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return response{Status: 1, Message: "Usage: list-invite <user>"}
	}
	var user User
	token := fields[0]
	if db.Where("token = ?", token).Preload("Invites").First(&user).RecordNotFound() {
		return response{Status: 1, Message: "Not login yet"}
	}
	var invites []string
	for _, invite := range user.Invites {
		invites = append(invites, invite.Username)
	}
	return response{Status: 0, Invite: invites}
}

func handleAcceptInvite(db *gorm.DB, cmdMsg string) response {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 2 {
		return response{Status: 1, Message: "Usage: Usage: accept-invite <user> <id>"}
	}
	var user User
	token, friendName := fields[0], fields[1]
	if db.Where("token = ?", token).Preload("Invites", "username = ?", friendName).First(&user).RecordNotFound() {
		return response{Status: 1, Message: "Not login yet"}
	}
	if len(user.Invites) == 0 {
		return response{Status: 1, Message: friendName + " did not invite you"}
	}
	var friend = user.Invites[0]
	db.Model(&user).Association("Invites").Delete(friend)
	db.Model(&user).Association("Friends").Append(friend)
	db.Model(&friend).Association("Friends").Append(user)
	return response{Status: 0, Message: "Success!"}
}

func handleListFriend(db *gorm.DB, cmdMsg string) response {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return response{Status: 1, Message: "Usage: list-friend <user>"}
	}
	var user User
	token := fields[0]
	if db.Where("token = ?", token).Preload("Friends").First(&user).RecordNotFound() {
		return response{Status: 1, Message: "Not login yet"}
	}
	var friends []string
	for _, friend := range user.Friends {
		friends = append(friends, friend.Username)
	}
	return response{Status: 0, Friend: friends}
}

func handlePost(db *gorm.DB, cmdMsg string) response {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fmt.Print(cmdMsg)
	fields := strings.SplitN(cmdMsg, " ", 2)
	if len(fields) != 2 {
		return response{Status: 1, Message: "Usage: post <user> <message>"}
	}
	var user User
	token, message := fields[0], fields[1]
	if db.Where("token = ?", token).First(&user).RecordNotFound() {
		return response{Status: 1, Message: "Not login yet"}
	}
	db.Create(&Post{OwnerID: user.ID, Message: message})
	return response{Status: 0, Message: "Success!"}
}

func handleRecivePost(db *gorm.DB, cmdMsg string) response {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fmt.Print(cmdMsg)
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return response{Status: 1, Message: "Usage: receive-post <user>"}
	}
	var user User
	token := fields[0]
	if db.Where("token = ?", token).Preload("Friends").First(&user).RecordNotFound() {
		return response{Status: 1, Message: "Not login yet"}
	}
	var friends []uint
	for _, friend := range user.Friends {
		friends = append(friends, friend.ID)
	}
	var posts []Post
	db.Where("owner_id in (?)", friends).Preload("Owner").Find(&posts)
	fmt.Println(posts)
	var responsePosts []responsePost
	for _, post := range posts {
		responsePosts = append(responsePosts, responsePost{ID: post.Owner.Username, Message: post.Message})
	}
	return response{Status: 0, Post: responsePosts}
}
