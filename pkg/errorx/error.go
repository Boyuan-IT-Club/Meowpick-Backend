package errorx

import (
	"fmt"
	"strings"

	internal2 "github.com/xh-polaris/innospark-core-api/pkg/errorx/internal"
)

// StatusError 是一个包含状态码的error接口, 可以使用New或WrapByCode创建
// 一个error并通过FromStatusError将它转换会StatusError来获取状态码等信息
type StatusError interface {
	error
	Code() int32
	Msg() string
	IsAffectStability() bool
	Extra() map[string]string
}

// Option 用来配置一个StatusError
type Option = internal2.Option

func KV(k, v string) Option {
	return internal2.Param(k, v)
}

func KVf(k, v string, a ...any) Option {
	formatValue := fmt.Sprintf(v, a...)
	return internal2.Param(k, formatValue)
}

func Extra(k, v string) Option {
	return internal2.Extra(k, v)
}

// New get an error predefined in the configuration file by statusCode
// with a stack trace at the point New is called.
func New(code int32, options ...Option) error {
	return internal2.NewByCode(code, options...)
}

// WrapByCode returns an error annotating err with a stack trace
// at the point WrapByCode is called, and the status code.
func WrapByCode(err error, statusCode int32, options ...Option) error {
	if err == nil {
		return nil
	}

	return internal2.WrapByCode(err, statusCode, options...)
}

// Wrapf returns an error annotating err with a stack trace
// at the point Wrapf is called, and the format specifier.
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	return internal2.Wrapf(err, format, args...)
}

func ErrorWithoutStack(err error) string {
	if err == nil {
		return ""
	}
	errMsg := err.Error()
	index := strings.Index(errMsg, "stack=")
	if index != -1 {
		errMsg = errMsg[:index]
	}
	return errMsg
}
