package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// CreateComment .
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

// GetCourseComments .
// @router /api/comment/query [GET]
func GetCourseComments(c *gin.Context) {
	// TODO 修改前端代码的调用 原本是"controller.query"需要改为GetCourseComments
	// TODO 注册router
	var err error
	var req cmd.GetCourseCommentsReq
	var resp *cmd.GetCommentsResp

	if err = c.ShouldBindQuery(&req); err != nil {
		common.PostProcess(c, &req, nil, err)
		return
	}

	if req.PageParam == nil {
		req.PageParam = &cmd.PageParam{} // 这里仅是防止空指针造成panic.{}可以留空，由之后的UnWrap方法设置默认值
		log.CtxInfo(c, "获得课程评论请求时PageParam为空，已设为默认值！")
	}

	c.Set(consts.ContextUserID, token.GetUserId(c))

	resp, err = provider.Get().CommentService.GetCourseComments(c, &req)
	common.PostProcess(c, &req, resp, err)
}

// GetTotalCommentsCount .
// @router /api/search/total [GET]
func GetTotalCommentsCount(c *gin.Context) {
	var err error
	var resp *cmd.GetTotalCommentsCountResp

	resp, err = provider.Get().CommentService.GetTotalCommentsCount(c)
	common.PostProcess(c, nil, resp, err)
}
