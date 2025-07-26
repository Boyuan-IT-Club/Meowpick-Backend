package comment

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Comment struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"    json:"id"`
	UserID   primitive.ObjectID `bson:"userId"           json:"userId"`
	CourseID primitive.ObjectID `bson:"courseId"         json:"courseId"`
	Content  string             `bson:"content"          json:"content"`
	Tags     []string           `bson:"tags"             json:"tags"`
	// Edited   bool               `bson:"edited"           json:"edited"`
	Deleted   bool      `bson:"deleted"          json:"-"` // 软删除标记通常不在API中返回
	CreatedAt time.Time `bson:"createdAt"        json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"        json:"updatedAt"`
}
