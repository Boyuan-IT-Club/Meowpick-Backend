package main

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/controller"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

func registerRoutes(router *gin.Engine) {
	commentGroup := router.Group("/api/comment")
	{
		commentGroup.POST("/add", controller.CreateComment)
	}
}

func Init() {
	provider.Init()
	log.Info("所有模块初始化完成...")
}

func main() {
	Init()
	router := gin.Default()

	registerRoutes(router)

	log.Info("服务器即将启动于 :8080")
	err := router.Run(":8080")
	if err != nil {
		panic(err)
	}
}
