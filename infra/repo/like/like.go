package like

import (
	"time"
)

type Like struct {
	ID         string    `bson:"_id,omitempty"  json:"id"`
	UserID     string    `bson:"userId"         json:"userId"`
	TargetID   string    `bson:"targetId"       json:"targetId"`
	TargetType int32     `bson:"targetType"     json:"targetType"`
	Active     bool      `bson:"active"         json:"active"`
	CreatedAt  time.Time `bson:"createdAt"      json:"createdAt"`
	UpdatedAt  time.Time `bson:"updatedAt"      json:"updatedAt"`
}
