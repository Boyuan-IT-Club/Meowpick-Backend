// Copyright 2025 Boyuan-IT-Club
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	err := r.Run(":8080")
	if err != nil {
		panic(err)
	}
}
