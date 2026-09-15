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

package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

var config *Config

type Auth struct {
	SecretKey    string
	PublicKey    string
	AccessExpire int64
}

type WeApp struct {
	AppID     string
	AppSecret string
}

type DebugLogin struct {
	Enabled    bool
	VerifyCode string
	OpenID     string
}

type Config struct {
	service.ServiceConf
	ListenOn string
	State    string
	Auth     Auth
	Mongo    struct {
		URL string
		DB  string
	}
	Cache         cache.CacheConf
	Redis         *redis.RedisConf
	WeApp         WeApp
	DebugLogin    DebugLogin
	AdminGrantKey string
}

func NewConfig() (*Config, error) {
	c := new(Config)
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "etc/config.yaml"
	}
	err := conf.Load(path, c)
	if err != nil {
		return nil, err
	}
	err = c.SetUp()
	if err != nil {
		return nil, err
	}
	if err = validateWeApp(c.WeApp); err != nil {
		return nil, err
	}
	if err = validateDebugLogin(c.State, c.DebugLogin); err != nil {
		return nil, err
	}
	config = c
	return c, nil
}

func validateWeApp(weApp WeApp) error {
	missing := make([]string, 0, 2)
	if strings.TrimSpace(weApp.AppID) == "" {
		missing = append(missing, "WeApp.AppID")
	}
	if strings.TrimSpace(weApp.AppSecret) == "" {
		missing = append(missing, "WeApp.AppSecret")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	return nil
}

func validateDebugLogin(state string, debugLogin DebugLogin) error {
	if !debugLogin.Enabled {
		return nil
	}
	if state != "local" {
		return fmt.Errorf("DebugLogin may only be enabled when State is local")
	}

	missing := make([]string, 0, 2)
	if strings.TrimSpace(debugLogin.VerifyCode) == "" {
		missing = append(missing, "DebugLogin.VerifyCode")
	}
	if strings.TrimSpace(debugLogin.OpenID) == "" {
		missing = append(missing, "DebugLogin.OpenID")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	return nil
}

func GetConfig() *Config {
	return config
}
