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
	}

	// SearchApi
	searchGroup := router.Group("/api/search")
	{
		searchGroup.GET("/recent", controller.GetSearchHistory)
		searchGroup.POST("", controller.LogSearch)
		searchGroup.GET("/total", controller.GetTotalCommentsCount)
	}
	// AuthApi
	authGroup := router.Group("")
	authGroup.POST("/sign_in", controller.SignIn)

	// LikeApi
	likeGroup := router.Group("/api/action")
	likeGroup.POST("/like/:id", controller.Like)
	return router
}
