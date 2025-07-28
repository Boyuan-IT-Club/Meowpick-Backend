package util

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/golang-jwt/jwt/v5" // 新增的go mod 依赖
	"time"
)

// GenerateJWTByPayload 用于将任意payload结构体作为JWT内容
func GenerateJWTByPayload(payload any) (string, error) {
	secret := config.GetConfig().Auth.SecretKey
	expire := time.Now().Add(time.Duration(config.GetConfig().Auth.AccessExpire) * time.Second)
	claims := jwt.MapClaims{
		"payload": JSONF(payload), // 用lib.go的JSONF序列化
		"exp":     expire.Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
