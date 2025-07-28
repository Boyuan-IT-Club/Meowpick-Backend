package authtoken

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/user"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
)

// AuthToken 认证Token类 字段和原项目保持一致
type AuthToken struct {
	MeowUser      user.User `json:"-"` // 不序列化到JSON
	NeedChangePwd bool      `json:"needChangePwd"`
	Session       string    `json:"session"`
	LastLoginInfo string    `json:"lastLoginInfo"`

	// JWT相关字段
	AccessToken string `json:"accessToken"`
	ExpiresAt   int64  `json:"expiresAt"` // Unix时间戳
}

// NewAuthorizedToken 创建认证成功的Token并返回JWT字符串
func NewAuthorizedToken(u *user.User, needChangePwd bool, session, lastLoginInfo string) (string, error) {
	token := AuthToken{
		MeowUser:      *u,
		NeedChangePwd: needChangePwd,
		Session:       session,
		LastLoginInfo: lastLoginInfo,
	}
	return util.GenerateJWTByPayload(token)
}
