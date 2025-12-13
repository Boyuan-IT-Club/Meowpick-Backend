package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

var _ ICommentCache = (*CommentCache)(nil)

const (
	CommentCacheKeyPrefix = "meowpick:comment:"
)

type ICommentCache interface {
	GetCount(ctx context.Context) (int64, bool, error)
	SetCount(ctx context.Context, count int64, ttl time.Duration) error
}

type CommentCache struct {
	cache *redis.Redis
}

func NewCommentCache(config *config.Config) *CommentCache {
	cache := redis.MustNewRedis(*config.Redis)
	return &CommentCache{cache: cache}
}

// GetCount 获取评论总数缓存
func (c *CommentCache) GetCount(ctx context.Context) (int64, bool, error) {
	countStr, err := c.cache.GetCtx(ctx, CommentCacheKeyPrefix+consts.CacheCommentCount)
	if err != nil {
		return 0, false, err
	}
	if countStr == "" {
		return 0, false, nil
	}
	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		_, _ = c.cache.DelCtx(ctx, CommentCacheKeyPrefix+consts.CacheCommentCount)
		return 0, false, err
	}
	return count, true, nil
}

// SetCount 设置评论总数缓存
func (c *CommentCache) SetCount(ctx context.Context, count int64, ttl time.Duration) error {
	return c.cache.SetexCtx(ctx, CommentCacheKeyPrefix+consts.CacheCommentCount,
		strconv.FormatInt(count, 10), int(ttl.Seconds()))
}
