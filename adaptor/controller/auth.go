package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/gin-gonic/gin"
)

// SignIn 用户登录接口
// @router /sign_in [POST]
func SignIn(c *gin.Context) {
	var err error
	var req cmd.SignInRequest

	if err = c.ShouldBindJSON(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	// 调用service
	p := provider.Get().AuthService
	resp, err := p.SignIn(c.Request.Context(), &req)
	common.PostProcess(c, &req, resp, err)
}

// 注册路由
func RegisterAuthRoutes(r *gin.Engine) {
	r.POST("/sign_in", SignIn)
}
