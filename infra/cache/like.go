package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

var _ ILikeCache = (*LikeCache)(nil)

type ILikeCache interface {
	// 点赞数相关
	GetLikeCount(ctx context.Context, targetID string) (int64, bool)
	GetBatchLikeCount(ctx context.Context, targetIDs []string, targetType int32) (map[string]int64, []string, error)
	SetLikeCount(ctx context.Context, targetID string, count int64, expiration time.Duration) error
	IncrLikeCount(ctx context.Context, targetID string, delta int64) (int64, error)
	DelLikeCount(ctx context.Context, targetID string) error

	// 点赞状态相关
	GetLikeStatus(ctx context.Context, userID, targetID string) (bool, bool)
	GetBatchLikeStatus(ctx context.Context, userID string, targetIDs []string, targetType int32) (map[string]bool, []string, error)
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

// GetBatchLikeCount 批量获取缓存，返回命中的id->LikeCount映射和未命中id列表
// 未命中id列表应作为mapper层聚合查询参数和SetBatchLikeCount的缓存键
func (r *LikeCache) GetBatchLikeCount(ctx context.Context, targetIDs []string, targetType int32) (map[string]int64, []string, error) {
	if len(targetIDs) == 0 {
		return make(map[string]int64), nil, nil
	}

	// 构建所有的缓存键
	keys := make([]string, len(targetIDs))
	for i, targetID := range targetIDs {
		keys[i] = "like:count:" + targetID
	}

	// 使用 MGET 批量获取
	values, err := r.client.Mget(keys...)
	if err != nil {
		return nil, targetIDs, err // 缓存失败，返回所有ID作为未命中
	}

	result := make(map[string]int64) // 命中的结果
	var missingIDs []string          // 未命中id列表
	// 遍历批量获取到的缓存结果
	for i, val := range values {
		targetID := targetIDs[i]
		if val == "" {
			// 缓存未命中，将id加入missingIDs
			missingIDs = append(missingIDs, targetID)
		} else {
			// 缓存命中，解析数值
			count, parseErr := strconv.ParseInt(val, 10, 64)
			if parseErr != nil {
				// 解析失败，视为未命中
				missingIDs = append(missingIDs, targetID)
			} else {
				// 缓存命中且解析成功，加入result
				result[targetID] = count
			}
		}
	}

	return result, missingIDs, nil
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

func (r *LikeCache) GetBatchLikeStatus(ctx context.Context, userID string, targetIDs []string, targetType int32) (map[string]bool, []string, error) {
	if len(targetIDs) == 0 {
		return make(map[string]bool), nil, nil
	}

	// 构建所有的缓存键
	keys := make([]string, len(targetIDs))
	for i, targetID := range targetIDs {
		keys[i] = "like:status:" + userID + ":" + targetID
	}

	// 使用 MGET 批量获取
	values, err := r.client.Mget(keys...)
	if err != nil {
		return nil, targetIDs, err // MGET失败，返回所有ID作为未命中
	}

	result := make(map[string]bool)
	var missingIDs []string

	for i, val := range values {
		targetID := targetIDs[i]
		if val == "" {
			// 缓存未命中
			missingIDs = append(missingIDs, targetID)
		} else {
			// 缓存命中，解析布尔值
			liked, parseErr := strconv.ParseBool(val)
			if parseErr != nil {
				// 解析失败，视为未命中
				missingIDs = append(missingIDs, targetID)
			} else {
				result[targetID] = liked
			}
		}
	}

	return result, missingIDs, nil
}
