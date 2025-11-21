package internal

import (
	"fmt"
)

// 携带错误信息和造成原因的错误
type withMessage struct {
	cause error
	msg   string
}

// Unwrap 拆包获取内部错误
func (w *withMessage) Unwrap() error {
	return w.cause
}

// Error 返回错误信息
func (w *withMessage) Error() string {
	return fmt.Sprintf("%s\ncause=%s", w.msg, w.cause.Error())
}

func wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	err = &withMessage{
		cause: err,
		msg:   fmt.Sprintf(format, args...),
	}

	return err
}

// Wrapf 使用格式化字符串包装一个错误
func Wrapf(err error, format string, args ...interface{}) error {
	return withStackTraceIfNotExists(wrapf(err, format, args...))
}
