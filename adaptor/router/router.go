package router

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/controller"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// SetupRoutes 函数，负责初始化所有路由
func SetupRoutes(p *provider.Provider) *gin.Engine {
	router := gin.Default()

	courseGroup := router.Group("/api/course") // 定义课程模块的路由组前缀
	{
		// 将 GET /api/course/list 请求，交给 CourseController 的 ListCourses 方法处理
		courseGroup.GET("/list", controller.ListCourses)
		// 以后如果有“添加课程”的功能，可以再加一行
	}

	return router
}
