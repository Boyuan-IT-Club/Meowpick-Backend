package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
	"net/http"
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
	p := provider.Get()

	resp, err = p.SearchHistoryService.LogSearch(c, req.Query)
	common.PostProcess(c, &req, resp, err)
}

// GetSearchHistory .
// @router /api/search/recent [GET]
func GetSearchHistory(c *gin.Context) {
	userID := token.GetUserId(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not logged in"})
		return
	}

	c.Set(consts.ContextUserID, userID)

	p := provider.Get()
	resp, err := p.SearchHistoryService.GetSearchHistoryByUserId(c)
	if err != nil {
		log.CtxError(c, "Service GetRecentByUserID failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get search history"})
		return
	}

	c.JSON(http.StatusOK, resp)
}
