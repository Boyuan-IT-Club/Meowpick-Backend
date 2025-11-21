package dto

import "time"

type CommentVO struct {
	ID       string   `json:"id"`
	CourseID string   `json:"courseId"`
	Content  string   `json:"content"`
	UserID   string   `json:"userId"`
	Tags     []string `json:"tags"`
	*LikeVO
	ExtraInfo
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ExtraInfo struct {
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Department string   `json:"department"`
	Teachers   []string `json:"teachers"` // 这里还是直接返回教师名字+职称组合后的字符串列表
}

// CreateCommentReq 对应 /api/comment/add 的请求体
type CreateCommentReq struct {
	CourseID string   `json:"courseId" binding:"required"`
	Content  string   `json:"content" binding:"required"`
	Tags     []string `json:"tags"`
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
	ID string `form:"id" binding:"required"` // TODO确定前端传来_id还是id
	*PageParam
}

// GetCourseCommentsResp 是后端返回给前端的、分页的评论历史数据。
type GetCourseCommentsResp struct {
	*Resp
	Total    int64        `json:"total"`
	Comments []*CommentVO `json:"comments"`
}

// GetMyCommentsResp “我的吐槽” 比一般的CommentVO多了一些课程的信息
type GetMyCommentsResp struct {
	*Resp
	Total    int64        `json:"total"`
	Comments []*CommentVO `json:"comments"`
}
