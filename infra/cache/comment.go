package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

var _ ICommentCache = (*CommentCache)(nil)

const (
	CommentCountCacheKey = "meowpick:comment:count"
)

type ICommentCache interface {
	GetCount(ctx context.Context) (int64, bool, error)
	SetCount(ctx context.Context, count int64, ttl time.Duration) error
}

type CommentCache struct {
	cache *redis.Redis
}

func NewCommentCache(cfg *config.Config) *CommentCache {
	cache := redis.MustNewRedis(*cfg.Redis)
	return &CommentCache{cache: cache}
}

// GetCount 获取评论总数缓存
func (c *CommentCache) GetCount(ctx context.Context) (int64, bool, error) {
	countStr, err := c.cache.GetCtx(ctx, CommentCountCacheKey)
	if err != nil {
		return 0, false, err
	}
	if countStr == "" {
		return 0, false, nil
	}
	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		_, _ = c.cache.DelCtx(ctx, CommentCountCacheKey)
		return 0, false, err
	}
	return count, true, nil
}

// SetCount 设置评论总数缓存
func (c *CommentCache) SetCount(ctx context.Context, count int64, ttl time.Duration) error {
	return c.cache.SetexCtx(ctx, CommentCountCacheKey, strconv.FormatInt(count, 10), int(ttl.Seconds()))
}
