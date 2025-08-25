package course

import (
	"time"
)

type Course struct {
	ID         string    `bson:"_id,omitempty"      json:"id"`
	Name       string    `bson:"name"               json:"name"`
	Code       string    `bson:"code"               json:"code"`
	TeacherIDs []string  `bson:"teacherIds"         json:"teacherIds"`
	Department int32     `bson:"department"         json:"department"`
	Category   int32     `bson:"category" json:"category"`
	Campuses   []int32   `bson:"campuses"           json:"campuses"`
	CreatedAt  time.Time `bson:"createdAt"          json:"createdAt"`
	UpdatedAt  time.Time `bson:"updatedAt"          json:"updatedAt"`
}
