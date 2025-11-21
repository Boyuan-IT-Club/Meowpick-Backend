package code

import (
	"github.com/xh-polaris/innospark-core-api/pkg/errorx/internal"
)

type RegisterOptionFn = internal.RegisterOption

// WithAffectStability 设置稳定性flag
// true:  会影响系统稳定性, 并反应在接口错误率上
// false: 不会影响稳定性
func WithAffectStability(affectStability bool) RegisterOptionFn {
	return internal.WithAffectStability(affectStability)
}

// Register 注册用户的预定义的错误代码, 并在初始化时调用PSM服务生成对应的子模块
func Register(code int32, msg string, opts ...RegisterOptionFn) {
	internal.Register(code, msg, opts...)
}

// SetDefaultErrorCode 带有PSM信息染色的Code被替换为默认code
func SetDefaultErrorCode(code int32) {
	internal.SetDefaultErrorCode(code)
}
