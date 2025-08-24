package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
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
	var resp *cmd.LogSearchResp

	if err = c.ShouldBindJSON(&req); err == nil {
		// TODO: 这里的 userID 将在未来由“认证中间件”提供
		userID := "64400e63eaf657ecc88324d4"

		p := provider.Get()
		resp, err = p.SearchHistoryService.LogSearch(c.Request.Context(), userID, req.Query)
	}

	common.PostProcess(c, &req, resp, err)
}

// GetSearchHistory .
// @router /api/search/recent [GET]
func GetSearchHistory(c *gin.Context) {
	// TODO: 这里的 userID 将在未来由“认证中间件”提供
	c.Set(consts.UserId, "64400e63eaf657ecc88324d4")
	userIDValue, exists := c.Get(consts.UserId)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not logged in"})
		return
	}

	userID, ok := userIDValue.(string)
	if !ok {
		log.CtxError(c.Request.Context(), "userID in context is not a string: %T", userIDValue)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	p := provider.Get()
	resp, err := p.SearchHistoryService.GetSearchHistoryByUserId(c, userID)
	if err != nil {
		log.CtxError(c, "Service GetRecentByUserID failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get search history"})
		return
	}

	c.JSON(http.StatusOK, resp)
}
