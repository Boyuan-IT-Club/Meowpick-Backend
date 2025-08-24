package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
	"net/http"
)

// CreateComment .
// @router /api/comment/add [POST]
func CreateComment(c *gin.Context) {
	var err error
	var req cmd.CreateCommentReq
	var resp *cmd.CreateCommentResp

	if err = c.ShouldBindJSON(&req); err == nil {
		// TODO: 这里的 userID 将在未来由“认证中间件”提供
		userID := "66a0d533722904b3952243d4"
		resp, err = provider.Get().CommentService.CreateComment(c, &req, userID)
	}
	common.PostProcess(c, req, resp, err)
}

// GetTotalCommentsCount .
// @router /api/search/total [GET]
func GetTotalCommentsCount(c *gin.Context) {
	p := provider.Get()
	total, err := p.CommentService.GetTotalCommentsCount(c.Request.Context())
	if err != nil {
		log.CtxError(c.Request.Context(), "Service GetTotalCommentCount failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get total count"})
		return
	}

	c.JSON(http.StatusOK, total)
}
