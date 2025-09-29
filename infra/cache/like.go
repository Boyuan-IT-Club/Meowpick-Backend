package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ILikeCache interface {
	// 点赞数相关
	GetLikeCount(ctx context.Context, targetID string) (int64, bool)
	SetLikeCount(ctx context.Context, targetID string, count int64, expiration time.Duration) error
	IncrLikeCount(ctx context.Context, targetID string, delta int64) (int64, error)
	DelLikeCount(ctx context.Context, targetID string) error

	// 点赞状态相关
	GetLikeStatus(ctx context.Context, userID, targetID string) (bool, bool)
	SetLikeStatus(ctx context.Context, userID, targetID string, liked bool, expiration time.Duration) error
	DelLikeStatus(ctx context.Context, userID, targetID string) error
}

type LikeCache struct {
	client *redis.Redis
}

func NewLikeCache(config *config.Config) *LikeCache {
	rds := redis.MustNewRedis(*config.Redis)
	return &LikeCache{
		client: rds,
	}
}

// GetLikeCount 点赞数缓存操作
func (r *LikeCache) GetLikeCount(ctx context.Context, targetID string) (int64, bool) {
	key := "like:count:" + targetID
	val, err := r.client.Get(key)
	if err != nil {
		return 0, false
	}
	count, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false
	}
	return count, true
}

func (r *LikeCache) SetLikeCount(ctx context.Context, targetID string, count int64, expiration time.Duration) error {
	key := "like:count:" + targetID
	return r.client.Setex(key, strconv.FormatInt(count, 10), int(expiration.Seconds()))
}

func (r *LikeCache) IncrLikeCount(ctx context.Context, targetID string, delta int64) (int64, error) {
	key := "like:count:" + targetID
	if delta == 1 {
		return r.client.Incr(key)
	} else if delta == -1 {
		return r.client.Decr(key)
	} else {
		return r.client.Incrby(key, delta)
	}
}

func (r *LikeCache) DelLikeCount(ctx context.Context, targetID string) error {
	key := "like:count:" + targetID
	_, err := r.client.Del(key)
	return err
}

// 点赞状态缓存操作
func (r *LikeCache) GetLikeStatus(ctx context.Context, userID, targetID string) (bool, bool) {
	key := "like:status:" + userID + ":" + targetID
	val, err := r.client.Get(key)
	if err != nil {
		return false, false
	}
	liked, err := strconv.ParseBool(val)
	if err != nil {
		return false, false
	}
	return liked, true
}

func (r *LikeCache) SetLikeStatus(ctx context.Context, userID, targetID string, liked bool, expiration time.Duration) error {
	key := "like:status:" + userID + ":" + targetID
	return r.client.Setex(key, strconv.FormatBool(liked), int(expiration.Seconds()))
}

func (r *LikeCache) DelLikeStatus(ctx context.Context, userID, targetID string) error {
	key := "like:status:" + userID + ":" + targetID
	_, err := r.client.Del(key)
	return err
}
