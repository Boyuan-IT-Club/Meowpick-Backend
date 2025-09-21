package router

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/controller"
	"github.com/gin-gonic/gin"
)

func SetupRoutes() *gin.Engine {
	router := gin.Default()

	// CommentApi
	commentGroup := router.Group("/api/comment")
	{
		commentGroup.POST("/add", controller.CreateComment)       // 发布评论
		commentGroup.GET("/query", controller.ListCourseComments) // 分页获取课程下的评论
		commentGroup.POST("/history", controller.GetMyComments)   // 获得我的吐槽
	}

	// SearchApi
	searchGroup := router.Group("/api/search")
	{
		searchGroup.GET("/recent", controller.GetSearchHistory)      // 搜索历史
		searchGroup.POST("", controller.ListCourses)                 // 模糊搜索展示课程列表
		searchGroup.POST("/teacher", controller.ListTeachers)        // 模糊搜索展示教师列表
		searchGroup.GET("/total", controller.GetTotalCommentsCount)  // 小程序初始化界面的总吐槽数
		searchGroup.GET("/suggest", controller.GetSearchSuggestions) // 用户输入搜索内容期间获得搜索建议
	}

	// AuthApi
	authGroup := router.Group("")
	authGroup.POST("/sign_in", controller.SignIn) // 初始化时的登录、授权

	// LikeApi
	likeGroup := router.Group("/api/action")
	likeGroup.POST("/like/:id", controller.Like) // 为评论点赞

	// CourseApi
	courseGroup := router.Group("/api/course")
	{
		courseGroup.GET("/:courseId", controller.GetOneCourse)         // 精确搜索某个课程
		courseGroup.GET("/departs", controller.GetCourseDepartments)   // 获得某课程的“所属部门”信息
		courseGroup.GET("/categories", controller.GetCourseCategories) // 获得某课程的“课程类型”信息
		courseGroup.GET("/campuses", controller.GetCourseCampuses)     // 获得某课程的“开设校区”信息
	}

	// TeacherApi
	//teacherGroup := router.Group("/api/teacher")
	//{
	//}
	return router
}
