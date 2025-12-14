package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

var _ ILikeCache = (*LikeCache)(nil)

const (
	LikeStatusCacheKey = consts.CacheLikeKeyPrefix + "status:"
)

type ILikeCache interface {
	GetStatusByUserIdAndTarget(ctx context.Context, userId, targetId string) (bool, bool, error)
	SetStatusByUserIdAndTarget(ctx context.Context, userId, targetId string, isLike bool, ttl time.Duration) error
}

type LikeCache struct {
	cache *redis.Redis
}

func NewLikeCache(cfg *config.Config) *LikeCache {
	cache := redis.MustNewRedis(*cfg.Redis)
	return &LikeCache{cache: cache}
}

// GetStatusByUserIdAndTarget 获取点赞状态缓存
// 返回值：isLike, isHit, error
func (c *LikeCache) GetStatusByUserIdAndTarget(ctx context.Context, userId, targetId string) (bool, bool, error) {
	key := LikeStatusCacheKey + userId + ":" + targetId
	statusStr, err := c.cache.GetCtx(ctx, key)
	if err != nil {
		return false, false, err
	}
	if statusStr == "" {
		return false, false, nil
	}
	isLike, err := strconv.ParseBool(statusStr)
	if err != nil {
		_, _ = c.cache.DelCtx(ctx, key)
		return false, false, err
	}
	return isLike, true, nil
}

// SetStatusByUserIdAndTarget 设置点赞状态缓存
func (c *LikeCache) SetStatusByUserIdAndTarget(ctx context.Context, userId, targetId string, isLike bool, ttl time.Duration) error {
	key := LikeStatusCacheKey + userId + ":" + targetId
	return c.cache.SetexCtx(ctx, key, strconv.FormatBool(isLike), int(ttl.Seconds()))
}
