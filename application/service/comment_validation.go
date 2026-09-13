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

package service

import (
	"strings"
	"unicode/utf8"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
)

const (
	commentContentMaxRunes = 140
	commentTagMaxCount     = 4
)

var allowedCommentTags = map[string]struct{}{
	"容易": {},
	"硬核": {},
	"避雷": {},
	"推荐": {},
	"严格": {},
	"快跑": {},
	"幽默": {},
	"枯燥": {},
}

func normalizeCommentInput(content string, tags []string) (string, []string, error) {
	content = strings.TrimSpace(content)
	if content == "" || !utf8.ValidString(content) || utf8.RuneCountInString(content) > commentContentMaxRunes {
		return "", nil, errorx.New(errno.ErrCommentInvalidContent)
	}
	if len(tags) > commentTagMaxCount {
		return "", nil, errorx.New(errno.ErrCommentInvalidTags)
	}

	normalizedTags := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if _, allowed := allowedCommentTags[tag]; !allowed {
			return "", nil, errorx.New(errno.ErrCommentInvalidTags)
		}
		if _, duplicate := seen[tag]; duplicate {
			return "", nil, errorx.New(errno.ErrCommentInvalidTags)
		}
		seen[tag] = struct{}{}
		normalizedTags = append(normalizedTags, tag)
	}

	return content, normalizedTags, nil
}
