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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/api/router"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/go-kit/logs"
)

func main() {
	provider.Init()
	r := router.SetupRoutes()
	setLogLevel()

	err := r.Run(":8080")
	if err != nil {
		panic(err)
	}
}

func setLogLevel() {
	level := config.GetConfig().Log.Level

	logs.Infof("log level: %s", level)
	switch level {
	case "trace":
		logs.SetLevel(logs.LevelTrace)
	case "debug":
		logs.SetLevel(logs.LevelDebug)
	case "info":
		logs.SetLevel(logs.LevelInfo)
	case "notice":
		logs.SetLevel(logs.LevelNotice)
	case "warn":
		logs.SetLevel(logs.LevelWarn)
	case "error":
		logs.SetLevel(logs.LevelError)
	case "fatal":
		logs.SetLevel(logs.LevelFatal)
	default:
		logs.SetLevel(logs.LevelInfo)
	}
}
