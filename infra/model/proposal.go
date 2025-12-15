package model

import (
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
)

type Proposal struct {
	ID        string        `bson:"_id,omitempty"      json:"id"`
	UserID    string        `bson:"userId"             json:"userId"`  // 提出Proposal的用户ID
	Title     string        `bson:"title"              json:"title"`   // 标题
	Content   string        `bson:"content"            json:"content"` // 描述的内容
	Deleted   bool          `bson:"deleted" json:"deleted"`            // 删除标记
	Course    *dto.CourseVO `bson:"course"             json:"course"`  // 课程信息，包含教师的ID（未创建不需要ID）
	CreatedAt time.Time     `bson:"createdAt"          json:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"          json:"updatedAt"`
}
