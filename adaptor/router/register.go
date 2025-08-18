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
	}

	searchGroup := router.Group("/api/search")
	{
		searchGroup.GET("/recent", controller.GetSearchHistory)
	}

	return router
}
