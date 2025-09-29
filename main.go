package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/router"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
)

func Init() {
	provider.Init()
	// 打印配置文件绝对路径
	configPath := "etc/config.yaml"
	absPath, _ := filepath.Abs(configPath)
	fmt.Println("配置文件绝对路径:", absPath)

	// 验证文件是否存在
	if _, err := os.Stat(configPath); err != nil {
		panic("配置文件不存在，请检查: " + absPath)
	}

	log.Info("所有模块初始化完成...")
}

func main() {
	Init()

	r := router.SetupRoutes()

	log.Info("服务器即将启动于 :8080")
	err := r.Run(":8080")
	if err != nil {
		panic(err)
	}
}
