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

package log

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
)

func getLoggerCtx(ctx context.Context) logx.Logger {
	return logx.WithContext(ctx).WithCallerSkip(1)
}

func getLogger() logx.Logger {
	return logx.WithCallerSkip(1)
}

func CtxInfo(ctx context.Context, format string, v ...any) {
	getLoggerCtx(ctx).Infof(format, v...)
}

func Info(format string, v ...any) {
	getLogger().Infof(format, v...)
}

func CtxError(ctx context.Context, format string, v ...any) {
	getLoggerCtx(ctx).Errorf(format, v...)
}

func Error(format string, v ...any) {
	getLogger().Errorf(format, v...)
}

func CondError(cond bool, format string, v ...any) {
	if cond {
		Error(format, v...)
	}
}

func CtxDebug(ctx context.Context, format string, v ...any) {
	getLoggerCtx(ctx).Debugf(format, v...)
}
