package user

import (
	"time"
)

type User struct {
	ID            string    `bson:"_id,omitempty"         json:"id"`
	Username      string    `bson:"username"              json:"username"`
	OpenId        string    `bson:"openId"                json:"-"`
	Avatar        string    `bson:"avatar,omitempty"      json:"avatar,omitempty"`
	Email         string    `bson:"email,omitempty"       json:"email,omitempty"`
	EmailVerified bool      `bson:"emailVerified"         json:"emailVerified"`
	Ban           bool      `bson:"ban"                   json:"-"`
	Admin         bool      `bson:"admin"                 json:"-"`
	CreatedAt     time.Time `bson:"createdAt"             json:"createdAt"`
	UpdatedAt     time.Time `bson:"updatedAt"             json:"updatedAt"`
}
