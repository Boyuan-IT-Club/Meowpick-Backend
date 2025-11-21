package handler

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// SignIn 用户登录接口
// @router /api/sign_in [POST]
func SignIn(c *gin.Context) {
	var err error
	var req dto.SignInReq
	var resp *dto.SignInResp
	// 参数校验
	if err = c.ShouldBindJSON(&req); err != nil {
		common.PostProcess(c, &req, nil, errorx.ErrInvalidParams)
		return
	}
	// 解析tokenString（可能为空）
	tokenStr, _ := token.ExtractToken(c.Request.Header)
	c.Set(consts.ContextUserID, tokenStr)

	// 调用service
	resp, err = provider.Get().AuthService.SignIn(c, &req)
	common.PostProcess(c, &req, resp, err)
}
