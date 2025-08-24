package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// Like .
// @router /api/action/like/{id} [POST]
func Like(c *gin.Context) {
	var req cmd.CreateLikeReq
	var resp *cmd.LikeResp
	var err error

	// 解析目标id和用户id
	req.TargetID = c.Param("id") // 前端采用路由匹配传参，直接解析即可

	c.Set(consts.ContextUserID, token.GetUserId(c))
	// 未来可能需要添加targetType解析
	resp, err = provider.Get().LikeService.Like(c, &req)
	common.PostProcess(c, nil, resp, err)
}
