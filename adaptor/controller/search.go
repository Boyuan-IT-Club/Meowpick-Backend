package controller

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/service"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
	"net/http"
)

type SearchHistoryController struct {
	SearchHistoryService service.ISearchHistoryService
}

// GetSearchHistory 负责处理 GET /api/search/recent 的请求
func GetSearchHistory(c *gin.Context) {
	c.Set(consts.UserId, "64400e63eaf657ecc88324d4") // 假设这是已登录用户的ID
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
