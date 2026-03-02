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
	"net/http"
	"strings"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID               string `json:"userId"` // 业务系统用户ID
	jwt.RegisteredClaims        // 标准字段（exp, iat, iss等）
	//DeviceID             string `json:"deviceId"` // 设备标识（可选）
}

// NewAuthorizedToken 签发accessToken(jwt)
func NewAuthorizedToken(user *model.User) (string, error) {
	jwtConfig := config.GetConfig().Auth
	claims := Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * time.Duration(jwtConfig.AccessExpire))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "meowpick-auth",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtConfig.SecretKey))
}

// GetUserID 给Handler的接口，用于从jwt提取userId
func GetUserID(ctx *gin.Context) string {
	var token string
	var err error
	if token, err = ExtractToken(ctx.Request.Header); err != nil {
		logs.CtxErrorf(ctx, "[Token] [ExtractToken] error: %v", err)
		return ""
	}
	var claims *Claims
	if claims, err = Parse(token); err != nil {
		ctx.Set("tokenError", err)
		logs.CtxErrorf(ctx, "[Token] [Parse] error: %v", err)
		return ""
	}
	return claims.UserID
}

// ExtractToken 从Header中提取Token
func ExtractToken(header http.Header) (string, error) {
	authHeader := header.Get("Authorization")
	if authHeader == "" {
		logs.Error("[Token] [ExtractToken] error: no authorization header found")
		return "", errorx.New(errno.ErrAuthHeaderNotFound)
	}

	// 支持 Bearer token 或直接 token
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1], nil
	} else if len(parts) == 1 {
		return parts[0], nil
	}
	logs.Errorf("[Token] [ExtractToken] error: wrong token format: %s", authHeader)
	return "", errorx.New(errno.ErrAuthTokenFormatInvalid)
}

// ParseAndValidate 解析Token并验证有效性
func ParseAndValidate(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.GetConfig().Auth.SecretKey), nil
	})
	if err != nil {
		logs.Errorf("[JWT] [ParseWithClaims] error: %v", err)
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	logs.Errorf("[Token] [ParseAndValidate] invalid token: %s", tokenStr)
	return nil, errorx.New(errno.ErrAuthTokenInvalid)
}

// ShouldRenew 检查Token是否需要续期
func ShouldRenew(claims *Claims) bool {
	return time.Until(claims.ExpiresAt.Time) <= claims.ExpiresAt.Sub(claims.IssuedAt.Time)/2 // 剩余时间不足一半时续期
}

// Parse 基础Token解析，返回一个Claim指针
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
		logs.Errorf("[JWT] [ParseWithClaims] error: %v", err)
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		logs.Errorf("[Token] [Parse] invalid token: %s", tokenStr)
		return nil, errorx.New(errno.ErrAuthTokenInvalid)
	}
	return claims, nil
}
