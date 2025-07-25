package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

// 全局配置变量
var config *Config

type Auth struct {
	SecretKey    string `yaml:"SecretKey"`
	PublicKey    string `yaml:"PublicKey"`
	AccessExpire int64  `yaml:"AccessExpire"`
}

type Coze struct {
	Key   string `yaml:"Key"`
	BotId string `yaml:"BotId"`
}

type MongoConf struct {
	URL string `yaml:"URL"`
	DB  string `yaml:"DB"`
}

type RedisConf struct {
	Host string `yaml:"Host"`
	Type string `yaml:"Type"`
	Pass string `yaml:"Pass"`
}

type Config struct {
	Name     string    `yaml:"Name"`
	ListenOn string    `yaml:"ListenOn"`
	State    string    `yaml:"State"`
	Auth     Auth      `yaml:"Auth"`
	Mongo    MongoConf `yaml:"Mongo"`
	Redis    RedisConf `yaml:"Redis"`
	Coze     Coze      `yaml:"Coze"`
}

// NewConfig 从配置文件加载配置
func NewConfig() (*Config, error) {
	c := new(Config)

	// 查找配置文件路径
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "etc/config.yaml" // 默认路径
	}

	// 读取文件内容
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 使用 yaml.Unmarshal 解析到结构体
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return nil, err
	}

	config = c
	return c, nil
}

// GetConfig 获取全局配置实例
func GetConfig() *Config {
	return config
}
