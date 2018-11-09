package main

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
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
		result = handleRegister(db, cmdMsg)
	case "login":
		result = handleLogin(db, cmdMsg)
	case "delete":
		result = handleDelete(db, cmdMsg)
	case "logout":
		result = handleLogout(db, cmdMsg)
	case "invite":
		result = handleInvite(db, cmdMsg)
	case "list-invite":
		result = handleListInvite(db, cmdMsg)
	case "accept-invite":
		result = handleAcceptInvite(db, cmdMsg)
	case "list-friend":
		result = handleListFriend(db, cmdMsg)
	case "post":
		result = handlePost(db, cmdMsg)
	case "receive-post":
		result = handleRecivePost(db, cmdMsg)
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

func handleRegister(db *gorm.DB, cmdMsg string) *map[string]interface{} {
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

func handleLogin(db *gorm.DB, cmdMsg string) *map[string]interface{} {
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

func handleDelete(db *gorm.DB, cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return &map[string]interface{}{"status": 1, "message": "Usage: delete <user>"}
	}
	var user User
	token := fields[0]
	if db.Where("token = ?", token).First(&user).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": "Not login yet"}
	}
	db.Delete(Post{}, "owner_id = ?", user.ID)
	db.Exec("DELETE FROM friendships WHERE user_id = ? OR friend_id = ?", user.ID, user.ID)
	db.Exec("DELETE FROM invites WHERE user_id = ? OR friend_id = ?", user.ID, user.ID)
	db.Delete(&user)
	return &map[string]interface{}{"status": 0, "message": "Success!"}
}

func handleLogout(db *gorm.DB, cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return &map[string]interface{}{"status": 1, "message": "Usage: logout <user>"}
	}
	var user User
	token := fields[0]
	if db.Where("token = ?", token).First(&user).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": "Not login yet"}
	}
	user.Token = ""
	db.Save(&user)
	return &map[string]interface{}{"status": 0, "message": "Bye!"}
}

func handleInvite(db *gorm.DB, cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 2 {
		return &map[string]interface{}{"status": 1, "message": "Usage: invite <user> <id>"}
	}
	var user User
	token, friendName := fields[0], fields[1]
	if db.Where("token = ?", token).First(&user).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": "Not login yet"}
	}
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

func handleListInvite(db *gorm.DB, cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return &map[string]interface{}{"status": 1, "message": "Usage: list-invite <user>"}
	}
	var user User
	token := fields[0]
	if db.Where("token = ?", token).Preload("Invites").First(&user).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": "Not login yet"}
	}
	invites := make([]string, 0)
	for _, invite := range user.Invites {
		invites = append(invites, invite.Username)
	}
	return &map[string]interface{}{"status": 0, "invite": invites}
}

func handleAcceptInvite(db *gorm.DB, cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 2 {
		return &map[string]interface{}{"status": 1, "message": "Usage: Usage: accept-invite <user> <id>"}
	}
	var user User
	token, friendName := fields[0], fields[1]
	if db.Where("token = ?", token).Preload("Invites", "username = ?", friendName).First(&user).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": "Not login yet"}
	}
	if len(user.Invites) == 0 {
		return &map[string]interface{}{"status": 1, "message": friendName + " did not invite you"}
	}
	var friend = user.Invites[0]
	db.Model(&user).Association("Invites").Delete(friend)
	db.Model(&user).Association("Friends").Append(friend)
	db.Model(&friend).Association("Friends").Append(user)
	return &map[string]interface{}{"status": 0, "message": "Success!"}
}

func handleListFriend(db *gorm.DB, cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return &map[string]interface{}{"status": 1, "message": "Usage: list-friend <user>"}
	}
	var user User
	token := fields[0]
	if db.Where("token = ?", token).Preload("Friends").First(&user).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": "Not login yet"}
	}
	friends := make([]string, 0)
	for _, friend := range user.Friends {
		friends = append(friends, friend.Username)
	}
	return &map[string]interface{}{"status": 0, "friend": friends}
}

func handlePost(db *gorm.DB, cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.SplitN(cmdMsg, " ", 2)
	if len(fields) != 2 {
		return &map[string]interface{}{"status": 1, "message": "Usage: post <user> <message>"}
	}
	var user User
	token, message := fields[0], fields[1]
	if db.Where("token = ?", token).First(&user).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": "Not login yet"}
	}
	db.Create(&Post{OwnerID: user.ID, Message: message})
	return &map[string]interface{}{"status": 0, "message": "Success!"}
}

func handleRecivePost(db *gorm.DB, cmdMsg string) *map[string]interface{} {
	if res, ok := stupidToken(cmdMsg); !ok {
		return res
	}
	fields := strings.Fields(cmdMsg)
	if len(fields) != 1 {
		return &map[string]interface{}{"status": 1, "message": "Usage: receive-post <user>"}
	}
	var user User
	token := fields[0]
	if db.Where("token = ?", token).Preload("Friends").First(&user).RecordNotFound() {
		return &map[string]interface{}{"status": 1, "message": "Not login yet"}
	}
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
