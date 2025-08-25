package main

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/router" // <-- 1. 导入你的新 router 包
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
)

func Init() {
	provider.Init()
	log.Info("所有模块初始化完成...")
}

func main() {
	Init()
	log.Info("服务器即将启动于 :8080")
	r := router.SetupRoutes(provider.Get())
	err := r.Run(":8080")
	if err != nil {
		panic(err)
	}
}
