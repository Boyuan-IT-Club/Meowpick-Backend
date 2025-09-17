package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// ListCoursesByTeacher 分页获取教师开设的课程
// @router /api/teacher/query [GET]
func ListCoursesByTeacher(c *gin.Context) {
	var req *cmd.GetTeachersReq
	var resp *cmd.GetTeachersResp
	var err error
	if err = c.ShouldBindQuery(&req); err != nil {
		// 如果这里出错，err 就被赋值了。我们直接 return，
		// defer 会自动捕获这个 err 并处理错误响应。
		return
	}

	resp, err = provider.Get().TeacherService.ListCoursesByTeacher(c, req)
	common.PostProcess(c, req, resp, err)
}

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