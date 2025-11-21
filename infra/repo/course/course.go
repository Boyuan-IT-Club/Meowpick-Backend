package course

import (
	"time"
)

type Course struct {
	ID            string          `bson:"_id,omitempty"      json:"id"`
	Name          string          `bson:"name"               json:"name"`
	Code          string          `bson:"code"               json:"code"`
	TeacherIDs    []string        `bson:"teacherIds"         json:"teacherIds"`
	Department    int32           `bson:"department"         json:"department"`
	Category      int32           `bson:"category" json:"category"`
	Campuses      []int32         `bson:"campuses"           json:"campuses"`
	LinkedCourses []*CourseInLink `bson:"link" json:"link"`
	CreatedAt     time.Time       `bson:"createdAt"          json:"createdAt"`
	UpdatedAt     time.Time       `bson:"updatedAt"          json:"updatedAt"`
}

// CourseInLink 具体课程卡片中，[相关课程]字段的链接
// 未来预计不再存储相关课程信息，故此处容忍存储id+name
type CourseInLink struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
}
