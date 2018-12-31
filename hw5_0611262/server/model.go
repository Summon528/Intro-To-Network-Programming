package main

import (
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
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

type Instance struct {
	ID string `gorm:"primary_key"`
}

func ConnectDB() *gorm.DB {
	dbURL := "nphw5:" + AWSPassword + "@tcp(" + DBHost + ")/nphw5?charset=utf8&parseTime=True&loc=Local"
	db, err := gorm.Open("mysql", dbURL)
	// db.LogMode(true)
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&User{}, &Post{}, &Group{}, &Instance{})
	return db
}
