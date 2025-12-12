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

// teacher: 104 000 000 ~ 104 999 999

const (
	ErrTeacherNotFound             = 104000001
	ErrTeacherGetSuggestionsFailed = 104000002
	ErrTeacherExist                = 104000003
	ErrTeacherExistsFailed         = 104000004
	ErrTeacherInsertFailed         = 104000005
	ErrTeacherFindFailed           = 104000006
)

func init() {
	code.Register(
		ErrTeacherNotFound,
		"teacher not found: {name}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrTeacherGetSuggestionsFailed,
		"get teacher suggestions failed: {keyword}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrTeacherExist,
		"teacher already exists: {name}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrTeacherExistsFailed,
		"failed to check teacher exists: {name}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrTeacherInsertFailed,
		"failed to insert teacher: {name}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrTeacherFindFailed,
		"failed to find teacher: {name}",
		code.WithAffectStability(false),
	)
}
