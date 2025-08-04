package cmd

import "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"

// CreateCommentCmd 对应 /api/comment/add 的请求体
type CreateCommentCmd struct {
	// OpenAPI 文档里的 `target` 字段，这里我们明确其为 CourseID
	CourseID string `json:"target" binding:"required"`
	// OpenAPI 文档里的 `text` 字段，即评论内容
	Content string `json:"text" binding:"required"`
	// 评论标签，例如：["推荐", "避雷"]
	Tags []string `json:"tags"`
}

type CreateCommentResp struct {
	Code             int    `json:"-"`
	Msg              string `json:"-"`
	*comment.Comment `json:"comment"`
}
