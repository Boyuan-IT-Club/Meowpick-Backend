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

// proposal: 106 000 000 ~ 106 999 999

const (
	ErrProposalFindFailed      = 106000001
	ErrProposalCvtFailed       = 106000002
	ErrProposalToggleFailed    = 106000003
	ErrProposalCountFailed     = 106000004
	ErrProposalGetStatusFailed = 106000005
	ErrProposalCourseFindInProposalFailed = 106000006
	ErrProposalCourseFindInCoursesFailed  = 106000007
	ErrProposalCourseFoundInProposals     = 106000008
	ErrProposalCourseFoundInCourses       = 106000009
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
	code.Register(
		ErrProposalFindFailed,
		"failed to find proposals",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalCvtFailed,
		"failed to convert proposal from {src} to {dst}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalToggleFailed,
		"failed to toggle vote status",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalCountFailed,
		"failed to get vote count by {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalGetStatusFailed,
		"failed to get vote status by {key}: {value}",
		code.WithAffectStability(false),
	)
}
