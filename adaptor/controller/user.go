package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	util2 "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// Login 用户登录接口
func Login(c *gin.Context) {

	code := util2.ObtainParameter(c, "code")
	// 获取code失败
	if code == "" {
		common.PostProcess(c.Request.Context(), c, nil, nil, errorx.ErrReqNoCode)
	}
	// 获得openid
	// TODO 实现通过config获取AppId和AppSecret
	openid, err := util2.GetWXOpenID("", "", code)
	if err != nil {
		common.PostProcess(c.Request.Context(), c, nil, nil, errorx.ErrFetchOpenIDFailed)
		return
	}
	// 构造LoginCMD传给service
	var req cmd.LoginCMD

	if err := c.ShouldBindJSON(&req); err != nil { // TODO 确保前端请求和LoginCMD字段能对应
		common.PostProcess(c.Request.Context(), c, &req, nil, err)
	}
	req.OpenID = openid

	// 调用service
	p := provider.Get().UserService
	resp, err := p.Login(c.Request.Context(), req)
	common.PostProcess(c.Request.Context(), c, &req, resp, err)
}

// 注册路由
func RegisterUserRoutes(r *gin.Engine) {
	r.POST("/account/login", Login)
}
