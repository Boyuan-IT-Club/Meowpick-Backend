// Copyright © 2025 univero. All rights reserved.
// Licensed under the GNU Affero General Public License v3 (AGPL-3.0).
// license that can be found in the LICENSE file.

package errorx

import (
	"errors"
	"fmt"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
)

const unknowCode = 999

// Errorx 是HTTP服务的业务异常
// 若返回Errorx给前端, 则HTTP状态码应该是200, 且响应体为Errorx内容
// 最佳实践:
// - 业务处理链路的末端使用Errorx, PostProcess处理后给出用户友好的响应
// - 预定义一些Errorx作为常量
// - 除却末端的Errorx外, 其余的error照常处理
type Errorx struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func New(code int, msg string) *Errorx {
	return &Errorx{
		Code: code,
		Msg:  msg,
	}
}

// Error 实现了error接口, 返回错误字符串
func (e Errorx) Error() string {
	return fmt.Sprintf("code=%d, msg=%s", e.Code, e.Msg)
}

// EndE 的作用是记录错误日志, 并返回一个与err相同的Errorx
func EndE(err error) error {
	log.Error("error: ", err)
	var ex Errorx
	if errors.As(err, &ex) {
		return ex
	}
	return &Errorx{Code: unknowCode, Msg: err.Error()}
}

// EndM 记录错误日志, 并返回一个自定义消息的Errorx
func EndM(err error, msg string) error {
	log.Error("error: ", msg)
	return &Errorx{Code: unknowCode, Msg: msg}
}

// EndX 记录错误日志, 并返回一个自定义消息和code的Errorx
func EndX(err error, code int, msg string) error {
	log.Error("error: ", msg)
	return &Errorx{Code: code, Msg: msg}
}

// 定义常量错误
// 认证相关
var (
	ErrTokenCreationFailed     = New(1001, "AuthToken creation failed")
	ErrReqNoCode               = New(1002, "no code in request")
	ErrFetchOpenIDFailed       = New(1003, "failed to fetch open id")
	ErrAuthTokenCreationFailed = New(1004, "auth token creation failed")
)

// 数据库相关
var (
	ErrUserNotFound     = New(2001, "User Not Found")
	ErrUserDuplicate    = New(2002, "User Duplicate")
	ErrUserInsertFailed = New(2003, "User Insert Failed")
	ErrFindFailed       = New(2004, "User Find Operation Failed")
)
