package course

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Course struct {
	ID         primitive.ObjectID   `bson:"_id,omitempty"      json:"id"`
	Name       string               `bson:"name"               json:"name"`
	Code       string               `bson:"code"               json:"code"`
	TeacherIDs []primitive.ObjectID `bson:"teacherIds"         json:"teacherIds"`
	Department int32                `bson:"department"         json:"department"`
	Categories int32                `bson:"categories"         json:"categories"`
	Campuses   []int32              `bson:"campuses"           json:"campuses"`
	CreatedAt  time.Time            `bson:"createdAt"          json:"createdAt"`
	UpdatedAt  time.Time            `bson:"updatedAt"          json:"updatedAt"`
}
