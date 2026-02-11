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

package errno

import "github.com/Boyuan-IT-Club/go-kit/errorx/code"

// changelog: 109 000 000 ~ 109 999 999
const (
	ErrChangeLogFindFailed        = 109000001
	ErrChangeLogCvtFailed         = 109000002
	ErrChangeLogInsertFailed      = 109000003
	ErrChangeLogTargetTypeInvalid = 109000004
	ErrChangeLogNotFound          = 109000005
	ErrChangeLogActionInvalid     = 109000006
	ErrChangeLogSourceInvalid     = 109000007
)

func init() {
	code.Register(
		ErrChangeLogFindFailed,
		"failed to find changelog: {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrChangeLogCvtFailed,
		"failed to convert changelog from {src} to {dst}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrChangeLogInsertFailed,
		"failed to insert changelog: {targetType}: {targetId}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrChangeLogTargetTypeInvalid,
		"invalid changelog target type: {targetType}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrChangeLogNotFound,
		"changelog not found: {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrChangeLogActionInvalid,
		"invalid changelog action: {action}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrChangeLogSourceInvalid,
		"invalid changelog update source: {updateSource}",
		code.WithAffectStability(false),
	)
}
