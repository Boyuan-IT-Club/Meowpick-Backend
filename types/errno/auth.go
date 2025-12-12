package errno

import "github.com/Boyuan-IT-Club/go-kit/errorx/code"

// auth: 106 000 000 ~ 106 999 999

const (
	ErrAuthHeaderNotFound      = 106000001
	ErrAuthTokenFormatInvalid  = 106000002
	ErrAuthTokenInvalid        = 106000003
	ErrAuthOpenIDEmpty         = 106000004
	ErrAuthTokenGenerateFailed = 106000005
)

func init() {
	code.Register(
		ErrAuthHeaderNotFound,
		"auth header not found",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrAuthTokenFormatInvalid,
		"auth token format invalid",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrAuthTokenInvalid,
		"auth token invalid",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrAuthOpenIDEmpty,
		"auth openid empty",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrAuthTokenGenerateFailed,
		"auth token generate failed",
		code.WithAffectStability(false),
	)
}
