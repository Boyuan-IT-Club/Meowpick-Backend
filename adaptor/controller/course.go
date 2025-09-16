package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// GetOneCourse 精确搜索一个课程，返回课程元信息
// @router /api/course/query/:courseID [GET]
func GetOneCourse(c *gin.Context) {
	var resp *cmd.CourseVO
	var err error

	resp, err = provider.Get().CourseService.GetOneCourse(c, c.Param("courseID"))
	common.PostProcess(c, nil, resp, err)
}

// @router /api/course/query [GET]// TODO 修正路由：传入keyword的模糊匹配接口应该是[POST]/api/search携带req参数type="course" 建议把此接口移到search.go
func GetCourses(ctx *gin.Context) {
	var req *cmd.GetCoursesReq
	var resp *cmd.GetCoursesResp
	var err error
	if err = ctx.ShouldBindQuery(&req); err != nil {
		// 如果这里出错，err 就被赋值了。我们直接 return，
		// defer 会自动捕获这个 err 并处理错误响应。
		return
	}

	resp, err = provider.Get().CourseService.ListCourses(ctx, req)
	common.PostProcess(ctx, req, resp, err)
}

// @router /api/course/departs [GET]
func GetCourseDepartments(ctx *gin.Context) {
	var req *cmd.GetCoursesDepartsReq
	var resp *cmd.GetCoursesDepartsResp
	var err error
	if err = ctx.ShouldBindQuery(&req); err != nil {
		return
	}
	resp, err = provider.Get().CourseService.GetDeparts(ctx, req)
	common.PostProcess(ctx, req, resp, err)
}

// GetCourseCategories xxx
// @router /api/course/categories [GET]
func GetCourseCategories(ctx *gin.Context) {
	var req *cmd.GetCourseCategoriesReq
	var resp *cmd.GetCourseCategoriesResp
	var err error
	if err = ctx.ShouldBindQuery(&req); err != nil {
		return
	}
	resp, err = provider.Get().CourseService.GetCategories(ctx, req)
	common.PostProcess(ctx, req, resp, err)
}

func GetCourseCampuses(ctx *gin.Context) {
	var req *cmd.GetCourseCampusesReq
	var resp *cmd.GetCourseCampusesResp
	var err error
	if err = ctx.ShouldBindQuery(&req); err != nil {
		return
	}
	resp, err = provider.Get().CourseService.GetCampuses(ctx, req)
	common.PostProcess(ctx, req, resp, err)
}
