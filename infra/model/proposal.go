package model

import (
	"time"
)

type Proposal struct {
	ID         string    `bson:"_id,omitempty" json:"id"`
	UserID     string    `bson:"userId"        json:"userId"`
	Title      string    `bson:"title"         json:"title"`
	Content    string    `bson:"content"       json:"content"`
	Deleted    bool      `bson:"deleted"       json:"deleted"`
	Course     Course    `bson:"course"        json:"course"`
	Status     int32     `bson:"status"        json:"status"` // pending / approved / rejected
	AgreeCount int64     `bson:"agreeCount"    json:"agreeCount"`
	CreatedAt  time.Time `bson:"createdAt"     json:"createdAt"`
	UpdatedAt  time.Time `bson:"updatedAt"     json:"updatedAt"`
}
