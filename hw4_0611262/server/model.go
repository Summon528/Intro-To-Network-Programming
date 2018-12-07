package main

import (
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type User struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Username  string `gorm:"not null;unique"`
	Password  string `gorm:"not null;"`
	Token     string
	Posts     []Post   `gorm:"foreignkey:Owner"`
	Friends   []*User  `gorm:"many2many:friendships;association_jointable_foreignkey:friend_id"`
	Invites   []*User  `gorm:"many2many:invites;association_jointable_foreignkey:friend_id"`
	Groups    []*Group `gorm:"many2many:group_members"`
}

type Post struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Message   string `gorm:"not null"`
	OwnerID   uint   `gorm:"not null"`
	Owner     User   `gorm:"foreignkey:OwnerID"`
}

type Group struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Groupname string
	Members   []*User `gorm:"many2many:group_members"`
}

func ConnectDB() *gorm.DB {
	db, err := gorm.Open("sqlite3", "db.sqlite")
	// db.LogMode(true)
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&User{}, &Post{}, &Group{})
	return db
}
