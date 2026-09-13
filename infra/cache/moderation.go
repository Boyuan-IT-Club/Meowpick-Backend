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

package cache

import (
	"context"
	"fmt"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/google/wire"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	commentModerationRateLimitKeyPrefix = "meowpick:moderation:comment:"
	commentModerationRateLimit          = int64(10)
	commentModerationRateWindowSeconds  = 60
)

const moderationRateLimitScript = `
local count = redis.call("INCR", KEYS[1])
if count == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return count
`

type IModerationRateLimiter interface {
	AllowComment(ctx context.Context, userID string) (bool, error)
}

type ModerationRateLimiter struct {
	cache *redis.Redis
}

var ModerationRateLimiterSet = wire.NewSet(
	NewModerationRateLimiter,
	wire.Bind(new(IModerationRateLimiter), new(*ModerationRateLimiter)),
)

func NewModerationRateLimiter(cfg *config.Config) *ModerationRateLimiter {
	return &ModerationRateLimiter{cache: redis.MustNewRedis(*cfg.Redis)}
}

func (l *ModerationRateLimiter) AllowComment(ctx context.Context, userID string) (bool, error) {
	result, err := l.cache.EvalCtx(
		ctx,
		moderationRateLimitScript,
		[]string{commentModerationRateLimitKeyPrefix + userID},
		commentModerationRateWindowSeconds,
	)
	if err != nil {
		return false, err
	}
	count, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected moderation rate-limit result type %T", result)
	}
	return count <= commentModerationRateLimit, nil
}
