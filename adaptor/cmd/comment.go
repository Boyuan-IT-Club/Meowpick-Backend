package cmd

import "time"

type CommentVO struct {
	ID       string   `json:"_id"`    // MongoDB _id
	CourseID string   `json:"target"` // courseId
	Content  string   `json:"text"`   // content
	UserID   string   `json:"uid"`    // 将后端 UserID 映射为 uid
	Tags     []string `json:"tags"`
	*LikeVO

	CreatedAt time.Time `json:"crateAt"`   // 注意拼写，映射为 crateAt
	UpdatedAt time.Time `json:"updatedAt"` // 注意拼写，映射为 updateAt
}

// CreateCommentReq 对应 /api/comment/add 的请求体
type CreateCommentReq struct {
	// OpenAPI 文档里的 `target` 字段，这里我们明确其为 CourseID
	CourseID string `json:"target" binding:"required"`
	// OpenAPI 文档里的 `text` 字段，即评论内容
	Content string   `json:"text" binding:"required"`
	Tags    []string `json:"tags"`
}

// CreateCommentResp 对应 /api/comment/add 的响应体
type CreateCommentResp struct {
	*Resp
	*CommentVO
}

// GetTotalCommentsCountResp 对应 /api/search/total 的响应体
type GetTotalCommentsCountResp struct {
	*Resp
	Count int64 `json:"count"`
}

// GetMyCommentsReq 是前端请求“我的吐槽”时，需要传递的数据结构。
type GetMyCommentsReq struct {
	*PageParam
}

// GetCourseCommentsReq 是前端分页请求某一课程下的评论时，需要传递的数据结构。
type GetCourseCommentsReq struct {
	CourseID string `form:"id" binding:"required"` // TODO确定前端传来_id还是id
	*PageParam
}

// GetCommentsResp 是后端返回给前端的、分页的评论历史数据。
type GetCommentsResp struct {
	*Resp
	Total int64        `json:"total"`
	Rows  []*CommentVO `json:"rows"`
}

// GetTotalCommentsCountResp 对应 /api/search/total 的响应体
type GetTotalCommentsCountResp struct {
	*Resp
	Count int64 `json:"count"`
}
