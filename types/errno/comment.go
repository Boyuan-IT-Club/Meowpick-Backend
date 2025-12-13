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

// comment: 105 000 000 ~ 105 999 999

const (
	ErrCommentInsertFailed = 105000001
	ErrCommentCvtFailed    = 105000002
	ErrCommentCountFailed  = 105000003
	ErrCommentFindFailed   = 105000004
)

func init() {
	code.Register(
		ErrCommentInsertFailed,
		"failed to insert comment: {content}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrCommentCvtFailed,
		"failed to convert comment from {src} to {dst}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrCommentCountFailed,
		"failed to count comments",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrCommentFindFailed,
		"failed to find comments by {key}: {value}",
		code.WithAffectStability(false),
	)
}
