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
	ErrProposalFindFailed                  = 108000001
	ErrProposalCvtFailed                   = 108000002
	ErrProposalToggleFailed                = 108000003
	ErrProposalCountFailed                 = 108000004
	ErrProposalGetStatusFailed             = 108000005
	ErrProposalCourseFindInProposalsFailed = 108000006
	ErrProposalCourseFindInCoursesFailed   = 108000007
	ErrProposalCourseFoundInProposals      = 108000008
	ErrProposalCourseFoundInCourses        = 108000009
	ErrProposalCreateFailed                = 108000010
	ErrProposalNotFound                    = 108000011
	ErrProposalIDRequired                  = 108000012
	ErrProposalAlreadyProcessed            = 108000013
	ErrProposalUpdateFailed                = 108000014
	ErrUserPermissionDenied                = 108000015
)

func init() {
	code.Register(
		ErrProposalCourseFindInProposalsFailed,
		"failed to find course in proposals: {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalCourseFindInCoursesFailed,
		"failed to find course in courses: {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalCourseFoundInProposals,
		"found course in proposals: {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalCourseFoundInCourses,
		"found course in courses: {key}: {value}",
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
		"failed to toggle proposal status",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalCountFailed,
		"failed to get proposal count by {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalGetStatusFailed,
		"failed to get proposal status by {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalCreateFailed,
		"failed to create proposal: {name}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalNotFound,
		"proposal not found: {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalIDRequired,
		"proposal ID is required: {key}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalAlreadyProcessed,
		"proposal already processed: {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrProposalUpdateFailed,
		"failed to update proposal: {proposalId}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrUserPermissionDenied,
		"user permission denied: {userId}",
		code.WithAffectStability(false),
	)
}
