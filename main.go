package main

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
)

func main() {
	// 初始化jwt-accessToken包
	token.Init(config.GetConfig())

}
