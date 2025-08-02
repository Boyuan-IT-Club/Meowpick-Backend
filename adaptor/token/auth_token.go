// Package token 签发、解析accessToken 提供：
// 用于提取user ID/OpenID接口
package token

import (
	"fmt"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/user"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"strings"
	"time"
)

var (
	jwtConfig *config.Auth
)

// Init 初始化token包（需在main中调用）
func Init(conf *config.Config) {
	jwtConfig = &conf.Auth
}

type Claims struct {
	UserID primitive.ObjectID `json:"userId"` // 业务系统用户ID
	OpenID string             `json:"openId"` // 微信OpenID
	//DeviceID             string `json:"deviceId"` // 设备标识（可选）
	jwt.RegisteredClaims // 标准字段（exp, iat, iss等）
}

// NewAuthorizedToken 签发accessToken(jwt)
func NewAuthorizedToken(user *user.User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		OpenID: user.OpenId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * time.Duration(jwtConfig.AccessExpire))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "meowpick-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtConfig.SecretKey))
}

// ExtractToken 从Header中提取Token
func ExtractToken(header http.Header) (string, error) {
	authHeader := header.Get("Authorization")
	if authHeader == "" {
		return "", errorx.ErrReqNoToken
	}

	// 支持 Bearer token 或直接token
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1], nil
	} else if len(parts) == 1 {
		return parts[0], nil
	}

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
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtConfig.SecretKey), nil
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

// ParseUserID 解析jwt中的user.ID 返回一个primitive.ObjectID对象
func ParseUserID(tokenStr *string) (UserID primitive.ObjectID, err error) {
	claims, err := Parse(*tokenStr)
	if err != nil {
		return UserID, err
	}
	return claims.UserID, nil
}

func ParseOpenID(tokenStr string) (string, error) {
	claims, err := Parse(tokenStr)
	if err != nil {
		return "", err
	}
	return claims.OpenID, nil
}
