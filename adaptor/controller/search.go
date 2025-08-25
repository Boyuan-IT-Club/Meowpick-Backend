package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// LogSearch .
// @router /api/search [POST]
func LogSearch(c *gin.Context) {
	var err error
	var req cmd.LogSearchReq
	var resp *cmd.Resp

	if err = c.ShouldBindJSON(&req); err != nil {
		common.PostProcess(c, &req, nil, err)
		return
	}

	userID := token.GetUserId(c)
	c.Set(consts.ContextUserID, userID)

	resp, err = provider.Get().SearchHistoryService.LogSearch(c, req.Query)
	common.PostProcess(c, &req, resp, err)
}

// GetSearchHistory .
// @router /api/search/recent [GET]
func GetSearchHistory(c *gin.Context) {
	var err error
	var resp *cmd.GetSearchHistoryResp

	c.Set(consts.ContextUserID, token.GetUserId(c))
	resp, err = provider.Get().SearchHistoryService.GetSearchHistoryByUserId(c)
	common.PostProcess(c, nil, resp, err)

	//userID := token.GetUserId(c)
	//if userID == "" {
	//	c.JSON(http.StatusUnauthorized, gin.H{"error": "user not logged in"})
	//	return
	//}
	//
	//c.Set(consts.ContextUserID, userID)
	//
	//resp, err := provider.Get().SearchHistoryService.GetSearchHistoryByUserId(c)
	//if err != nil {
	//	log.CtxError(c, "Service GetRecentByUserID failed: %v", err)
	//	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get search history"})
	//	return
	//}
	//
	//c.JSON(http.StatusOK, resp)

	//var err error
	//var req cmd.CreateCommentReq
	//var resp *cmd.CreateCommentResp
	//
	//if err = c.ShouldBindJSON(&req); err != nil {
	//	common.PostProcess(c, &req, nil, err)
	//	return
	//}
	//
	//c.Set(consts.ContextUserID, token.GetUserId(c))
	//resp, err = provider.Get().CommentService.CreateComment(c, &req)
	//common.PostProcess(c, &req, resp, err)
}
