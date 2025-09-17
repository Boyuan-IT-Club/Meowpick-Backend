package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// CreateComment 发布评论
// @router /api/comment/add [POST]
func CreateComment(c *gin.Context) {
	var err error
	var req cmd.CreateCommentReq
	var resp *cmd.CreateCommentResp

	if err = c.ShouldBindJSON(&req); err != nil {
		common.PostProcess(c, &req, nil, err)
		return
	}

	c.Set(consts.ContextUserID, token.GetUserId(c))
	resp, err = provider.Get().CommentService.CreateComment(c, &req)
	common.PostProcess(c, &req, resp, err)
}

// ListCourseComments 分页获取课程评论
// @router /api/comment/query [GET]
func ListCourseComments(c *gin.Context) {
	var err error
	var req cmd.GetCourseCommentsReq
	var resp *cmd.GetCommentsResp

	if err = c.ShouldBindQuery(&req); err != nil {
		common.PostProcess(c, &req, nil, err)
		return
	}

	if req.PageParam == nil {
		req.PageParam = &cmd.PageParam{} // 这里仅是防止空指针造成panic {}中留空，由之后的UnWrap方法设置默认值
		log.CtxInfo(c, "获得课程评论请求时PageParam为空，已设为默认值！")
	}

	c.Set(consts.ContextUserID, token.GetUserId(c))

	resp, err = provider.Get().CommentService.GetCourseComments(c, &req)
	common.PostProcess(c, &req, resp, err)
}

// GetTotalCommentsCount 获得小程序收录吐槽总数
// @router /api/search/total [GET]
func GetTotalCommentsCount(c *gin.Context) {
	var resp *cmd.GetTotalCommentsCountResp
	var err error

	resp, err = provider.Get().CommentService.GetTotalCommentsCount(c.Request.Context())
	common.PostProcess(c, nil, resp, err)
}

// GetMyComments .
// @router /api/comment/history [POST]
func GetMyComments(c *gin.Context) {
	var err error
	var req cmd.GetMyCommentsReq
	var resp *cmd.GetCommentsResp

	if err = c.ShouldBindJSON(&req); err != nil {
		common.PostProcess(c, &req, nil, err)
		return
	}

	c.Set(consts.ContextUserID, token.GetUserId(c))
	resp, err = provider.Get().CommentService.GetMyComments(c, &req)
	common.PostProcess(c, &req, resp, err)
}
