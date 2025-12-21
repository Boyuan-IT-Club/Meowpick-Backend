package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

var _ IProposalCache = (*ProposalCache)(nil)

const (
	ProposalStatusCacheKey = consts.CacheProposalKeyPrefix + "status"
)

type IProposalCache interface {
	GetStatusByUserIdAndTarget(ctx context.Context, userId, targetId string) (bool, bool, error)
	SetStatusByUserIdAndTarget(ctx context.Context, userId, targetId string, isVote bool, ttl time.Duration) error
}

type ProposalCache struct {
	cache *redis.Redis
}

func NewProposalCache(cfg *config.Config) *ProposalCache {
	cache := redis.MustNewRedis(*cfg.Redis)
	return &ProposalCache{cache: cache}
}

// GetStatusByUserIdAndTarget 获取点赞状态缓存
func (c *ProposalCache) GetStatusByUserIdAndTarget(ctx context.Context, userId, targetId string) (bool, bool, error) {
	key := ProposalStatusCacheKey + userId + ":" + targetId
	statusStr, err := c.cache.GetCtx(ctx, key)
	if err != nil {
		return false, false, nil
	}
	if statusStr == "" {
		return false, false, nil
	}
	isProposal, err := strconv.ParseBool(statusStr)
	if err != nil {
		_, _ = c.cache.DelCtx(ctx, key)
		return false, false, err
	}
	return isProposal, true, nil
}

// SetStatusByUserIdAndTarget 设置投票状态缓存
func (c *ProposalCache) SetStatusByUserIdAndTarget(ctx context.Context, userId, targetId string, isProposal bool, ttl time.Duration) error {
	key := ProposalStatusCacheKey + userId + ":" + targetId
	return c.cache.SetexCtx(ctx, key, strconv.FormatBool(isProposal), int(ttl.Seconds()))
}
