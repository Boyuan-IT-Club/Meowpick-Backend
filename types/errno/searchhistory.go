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

// search history: 103 000 000 ~ 103 999 999

const (
	ErrSearchHistoryFindFailed         = 103000001
	ErrSearchHistoryUpsertFailed       = 103000002
	ErrSearchHistoryDeleteOldestFailed = 103000003
	ErrSearchHistoryCountFailed        = 103000004
)

func init() {
	code.Register(
		ErrSearchHistoryFindFailed,
		"failed to find search history by {key}: {value}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrSearchHistoryUpsertFailed,
		"failed to upsert search history: {query}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrSearchHistoryDeleteOldestFailed,
		"failed to delete search history",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrSearchHistoryCountFailed,
		"failed to count search history",
		code.WithAffectStability(false),
	)
}
