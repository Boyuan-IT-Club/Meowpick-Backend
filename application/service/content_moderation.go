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
	"context"
	"errors"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/wechatsecurity"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
)

const contentModerationTimeout = 3 * time.Second

type IContentModerationService interface {
	CheckText(ctx context.Context, req wechatsecurity.TextCheckRequest) error
}

type ContentModerationService struct {
	Client wechatsecurity.IContentSecurityClient
}

var ContentModerationServiceSet = wire.NewSet(
	wire.Struct(new(ContentModerationService), "*"),
	wire.Bind(new(IContentModerationService), new(*ContentModerationService)),
)

func (s *ContentModerationService) CheckText(ctx context.Context, req wechatsecurity.TextCheckRequest) error {
	moderationCtx, cancel := context.WithTimeout(ctx, contentModerationTimeout)
	defer cancel()

	startedAt := time.Now()
	result, err := s.Client.CheckText(moderationCtx, req)
	elapsed := time.Since(startedAt)
	userID, _ := ctx.Value(consts.CtxUserID).(string)
	if err == nil && result == nil {
		err = errors.New("wechat content moderation returned no result")
	}
	if err != nil {
		logs.CtxWarnf(ctx,
			"[ContentModeration] unavailable, userId=%s scene=%d durationMs=%d error_type=%T",
			userID, req.Scene, elapsed.Milliseconds(), err,
		)
		return errorx.WrapByCode(err, errno.ErrContentModerationUnavailable)
	}

	logs.CtxInfof(ctx,
		"[ContentModeration] completed, userId=%s scene=%d suggest=%s label=%d traceId=%s durationMs=%d",
		userID, req.Scene, result.Suggest, result.Label, result.TraceID, elapsed.Milliseconds(),
	)
	switch result.Suggest {
	case "pass":
		return nil
	case "review", "risky":
		return errorx.New(errno.ErrContentModerationRejected)
	default:
		return errorx.New(errno.ErrContentModerationUnavailable)
	}
}
