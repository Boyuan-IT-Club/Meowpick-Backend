package changelog

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type ChangeLog struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"         json:"-"`
	TargetID     string             `bson:"targetId"              json:"-"`
	TargetType   int32              `bson:"targetType"            json:"-"`
	Action       int32              `bson:"action"                json:"-"`
	Content      string             `bson:"content"               json:"-"`
	UpdateSource int32              `bson:"updateSource"          json:"-"`
	ProposalID   string             `bson:"proposalId,omitempty"  json:"-"`
	UpdatedAt    time.Time          `bson:"updatedAt"             json:"-"`
}
