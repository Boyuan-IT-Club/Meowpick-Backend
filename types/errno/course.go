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

// course: 101 000 000 ~ 101 999 999

const (
	ErrCourseInvalidParam         = 101000001
	ErrCourseNotFound             = 101000002
	ErrCourseFindFailed           = 101000003
	ErrCourseCvtFailed            = 101000004
	ErrCourseGetSuggestionsFailed = 101000005
	ErrCourseGetDepartmentsFailed = 101000006
	ErrCourseGetCategoriesFailed  = 101000007
	ErrCourseGetCampusesFailed    = 101000008
	ErrCourseCreateFailed         = 101000009
)

func init() {
	code.Register(
		ErrCourseInvalidParam,
		"invalid parameter {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrCourseNotFound,
		"course not found by {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrCourseFindFailed,
		"failed to find course by {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrCourseCvtFailed,
		"failed to convert course from {src} to {dst}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrCourseGetSuggestionsFailed,
		"failed to get course suggestions: {keyword}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrCourseGetDepartmentsFailed,
		"failed to get course departments: {name}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrCourseGetCategoriesFailed,
		"failed to get course categories: {name}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrCourseGetCampusesFailed,
		"failed to get course campuses: {name}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrCourseCreateFailed,
		"failed to create course: {name}",
		code.WithAffectStability(false),
	)
}
