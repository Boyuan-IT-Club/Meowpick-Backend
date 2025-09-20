package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// AddNewTeacher 新建教师
// @router /api/teacher/add
func AddNewTeacher(c *gin.Context) {
	var req *cmd.AddNewTeacherReq
	var resp *cmd.AddNewTeacherResp
	var err error

	if err = c.ShouldBind(&req); err != nil {
		common.PostProcess(c, req, resp, err)
	}

	c.Set(consts.UserId, token.GetUserId(c))

	resp, err = provider.Get().TeacherService.AddNewTeacher(c, req)
	common.PostProcess(c, req, resp, err)
}
