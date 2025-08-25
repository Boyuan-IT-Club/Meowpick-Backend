package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// 此为/api/course/query
func GetCourses(ctx *gin.Context) {
	var query *cmd.CourseQueryCmd
	var resp *cmd.PaginatedCoursesResp
	var err error
	if err = ctx.ShouldBindQuery(&query); err != nil {
		// 如果这里出错，err 就被赋值了。我们直接 return，
		// defer 会自动捕获这个 err 并处理错误响应。
		return
	}

	util.CheckPage(query)
	resp, err = provider.Get().CourseService.ListCourses(ctx, *query)
	common.PostProcess(ctx, query, resp, err)
}
