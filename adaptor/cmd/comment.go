package cmd

import (
	"time"
)

// CreateCommentReq 对应 /api/comment/add 的请求体
type CreateCommentReq struct {
	// OpenAPI 文档里的 `target` 字段，这里我们明确其为 CourseID
	CourseID string `json:"target" binding:"required"`
	// OpenAPI 文档里的 `text` 字段，即评论内容
	Content string   `json:"text" binding:"required"`
	Tags    []string `json:"tags"`
	UserID  string   `json:"uid" binding:"required"`
}

type ResponseComment struct {
	ID       string   `json:"_id"`    // MongoDB _id
	CourseID string   `json:"target"` // courseId
	Content  string   `json:"text"`   // content
	UserID   string   `json:"uid"`    // 将后端 UserID 映射为 uid
	Tags     []string `json:"tags"`

	CreateAt time.Time `json:"crateAt"`  // 注意拼写，映射为 crateAt
	UpdateAt time.Time `json:"updateAt"` // 注意拼写，映射为 updateAt
}

// CreateCommentResp 对应 /api/comment/add 的响应体
type CreateCommentResp struct {
	Code    int              `json:"-"`
	Msg     string           `json:"-"`
	Comment *ResponseComment `json:"comment"`
}
