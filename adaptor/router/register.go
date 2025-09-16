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
		commentGroup.POST("/add", controller.CreateComment)
		commentGroup.GET("/query", controller.GetCourseComments)
		commentGroup.POST("/history", controller.GetMyComments)
	}

	// SearchApi
	searchGroup := router.Group("/api/search")
	{
		searchGroup.GET("/recent", controller.GetSearchHistory)
		searchGroup.POST("", controller.LogSearch)
		searchGroup.GET("/total", controller.GetTotalCommentsCount)
		searchGroup.GET("/suggest", controller.GetSearchSuggestions)
	}

	// AuthApi
	authGroup := router.Group("")
	authGroup.POST("/sign_in", controller.SignIn)

	// LikeApi
	likeGroup := router.Group("/api/action")
	likeGroup.POST("/like/:id", controller.Like)

	// CourseApi
	courseGroup := router.Group("/api/course")
	{
		courseGroup.GET("/:courseID", controller.GetOneCourse)
		courseGroup.GET("/departs", controller.GetCourseDepartments)
		courseGroup.GET("/categories", controller.GetCourseCategories)
		courseGroup.GET("/campuses", controller.GetCourseCampuses)
	}

	// TeacherApi
	teacherGroup := router.Group("/api/teacher")
	{
		teacherGroup.GET("/query", controller.GetCoursesByTeacher)
	}
	return router
}
