package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"os"

	"github.com/zeromicro/go-zero/core/service"

	"github.com/zeromicro/go-zero/core/conf"
)

var config *Config

type Auth struct {
	SecretKey    string
	PublicKey    string
	AccessExpire int64
}

//type SMTPConf struct {
//	Host     string `json:",env=SMTP_HOST"`
//	Port     int    `json:",default=587"`
//	Username string `json:",env=SMTP_USER"`
//	Password string `json:",env=SMTP_PASS"`
//	From     string `json:",default=no-reply@meowpick.com"`
//}
//
//type EmailVerifyConf struct {
//	ExpireSeconds int `json:",default=300"` // 验证码有效期(秒)
//	DailyLimit    int `json:",default=10"`  // 每日发送上限
//}

type Config struct {
	service.ServiceConf
	ListenOn string
	State    string
	Auth     Auth
	Mongo    struct {
		URL string
		DB  string
	}
	Cache       cache.CacheConf
	Redis       *redis.RedisConf
	WeAppSecret string
	//SMTP        SMTPConf
	//EmailVerify EmailVerifyConf
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
	config = c
	return c, nil
}

func GetConfig() *Config {
	return config
}
