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

// proposal: 108 000 000 ~ 108 999 999

const (
	ErrProposalCourseFindInProposalFailed = 108000001
	ErrProposalCourseFindInCoursesFailed  = 108000002
	ErrProposalCourseFoundInProposals     = 108000003
	ErrProposalCourseFoundInCourses       = 108000004
)

func init() {
	code.Register(
		ErrProposalCourseFindInProposalFailed,
		"failed to find course in proposal {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalCourseFindInCoursesFailed,
		"failed to find course in course {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalCourseFoundInProposals,
		"found course in proposal {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalCourseFoundInCourses,
		"found course in course {key}: {value}",
		code.WithAffectStability(false),
	)

}
