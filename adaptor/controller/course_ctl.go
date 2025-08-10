package controller

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/service"
	"github.com/gin-gonic/gin"
	"net/http" //获取状态码
)

type CourseController struct {
	courseSvc service.CourseService
}

func NewCourseController(courseSvc service.CourseService) *CourseController {
	return &CourseController{courseSvc: courseSvc}
}

// ctx (Context) 是 Gin 框架传入的，包含了所有请求和响应的信息
func (c *CourseController) ListCourses(ctx *gin.Context) {

	var query dto.CourseQuery
	// ShouldBindQuery会自动把 URL 中的 ?key=value 参数解析并填充到 query 这个结构体变量里
	if err := ctx.ShouldBindQuery(&query); err != nil {
		// 如果解析出错（比如用户传的参数类型不对），说明是客户端的错，返回 400 Bad Request
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的查询参数"})
		return
	}

	//验证，防止非法传参
	if query.Page <= 0 {
		query.Page = 1
	}
	// 给 PageSize 设置一个上限，防止一次请求过多数据拖垮服务器
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}

	// 这里传入了 ctx.Request.Context()，这是为了传递请求的上下文
	paginatedResult, err := c.courseSvc.ListCourses(ctx.Request.Context(), query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	//成功解析
	ctx.JSON(http.StatusOK, paginatedResult)
}
