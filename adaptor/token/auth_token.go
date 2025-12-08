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

// Package token 签发、解析accessToken 提供：
// 用于提取user ID/OpenID接口
package token

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/user"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"userId"` // 业务系统用户ID

	//DeviceID             string `json:"deviceId"` // 设备标识（可选）
	jwt.RegisteredClaims // 标准字段（exp, iat, iss等）
}

// NewAuthorizedToken 签发accessToken(jwt)
func NewAuthorizedToken(user *user.User) (string, error) {
	jwtConfig := config.GetConfig().Auth
	claims := Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * time.Duration(jwtConfig.AccessExpire))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "meowpick-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtConfig.SecretKey))
}

// GetUserId 给controller层的接口 用于从jwt提取userId
func GetUserId(ctx *gin.Context) string {
	var token string
	var err error

	if token, err = ExtractToken(ctx.Request.Header); err != nil {
		log.Error("ExtractToken Failed,err:", err)
		return ""
	}

	var claims *Claims
	if claims, err = Parse(token); err != nil {
		ctx.Set("tokenError", err)
		return ""
	}

	return claims.UserID
}

// ExtractToken 从Header中提取Token
func ExtractToken(header http.Header) (string, error) {
	authHeader := header.Get("Authorization")
	if authHeader == "" {
		log.Error("no Authorization header found")
		return "", errorx.ErrReqNoToken
	}

	// 支持 Bearer token 或直接token
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1], nil
	} else if len(parts) == 1 {
		return parts[0], nil
	}
	log.Error("no Bearer token found!Please check the Authorization field in the header")
	return "", errorx.ErrWrongTokenFmt
}

// ParseAndValidate 解析Token并验证有效性
func ParseAndValidate(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.GetConfig().Auth.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errorx.ErrTokenInvalid
}

// ShouldRenew 检查Token是否需要续期
func ShouldRenew(claims *Claims) bool {
	remaining := time.Until(claims.ExpiresAt.Time)
	total := claims.ExpiresAt.Sub(claims.IssuedAt.Time)
	return remaining <= total/2 // 剩余时间不足一半时续期
}

// Parse 基础Token解析 返回一个Claim指针
func Parse(tokenStr string) (*Claims, error) {
	secretKey := config.GetConfig().Auth.SecretKey

	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errorx.ErrTokenInvalid
	}

	return claims, nil
}

// ParseUserID 解析jwt中的user.ID 返回string类型
func ParseUserID(tokenStr *string) (UserID string, err error) {
	claims, err := Parse(*tokenStr)
	if err != nil {
		return UserID, err
	}
	return claims.UserID, nil
}
