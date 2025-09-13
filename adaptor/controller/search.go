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
}

// GetSearchSuggestions
// @router /api/search/suggest
func GetSearchSuggestions(c *gin.Context) {
	var err error
	var req *cmd.GetSearchSuggestReq
	var resp *cmd.GetSearchSuggestResp
	if err = c.ShouldBindQuery(&req); err != nil {
		common.PostProcess(c, req, nil, err)
		return
	}
	resp, err = provider.Get().SearchService.GetSearchSuggestions(c, req)

	log.Info("--- DEBUG [Controller]: About to call PostProcess ---\n")
	log.Info(">>> err variable is: %v\n", err)
	log.Info(">>> resp variable is: %#v\n", resp)

	common.PostProcess(c, req, resp, err)
}
