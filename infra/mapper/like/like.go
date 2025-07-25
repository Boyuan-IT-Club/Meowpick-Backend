package like

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Like struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"  json:"id"`
	UserID     primitive.ObjectID `bson:"userId"         json:"userId"`
	TargetID   primitive.ObjectID `bson:"targetId"       json:"targetId"`
	TargetType int32              `bson:"targetType"     json:"targetType"`
	Active     bool               `bson:"active"         json:"active"`
	CreatedAt  time.Time          `bson:"createdAt"      json:"createdAt"`
	UpdatedAt  time.Time          `bson:"updatedAt"      json:"updatedAt"`
}
