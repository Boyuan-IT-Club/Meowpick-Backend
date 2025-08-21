package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// LogSearch .
// @router /api/search [POST]
func LogSearch(c *gin.Context) {
	var err error
	var req cmd.LogSearchReq
	var resp *cmd.LogSearchResp

	if err = c.ShouldBindJSON(&req); err == nil {
		userID := "64400e63eaf657ecc88324d4" // 模拟用户ID

		p := provider.Get()
		resp, err = p.SearchHistoryService.LogSearch(c.Request.Context(), userID, req.Query)
	}

	common.PostProcess(c, &req, resp, err)
}
