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
	"strconv"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

var (
	DefaultTTL = 24 * time.Hour
)

type MappingCache struct {
	cache *redis.Redis
}

func NewMappingCache(cfg *config.Config) *MappingCache {
	cache := redis.MustNewRedis(*cfg.Redis)
	return &MappingCache{cache: cache}
}

func (c *MappingCache) GetCodeByKey(ctx context.Context, mappingType model.MappingType, name string) (int32, bool, error) {
	key := fmt.Sprintf("mapping:code:%d:%s", mappingType, name)
	val, err := c.cache.GetCtx(ctx, key)
	if err != nil {
		return 0, false, err
	}
	if val == "" {
		return 0, false, nil
	}
	code, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return 0, false, err
	}
	return int32(code), true, nil
}

func (c *MappingCache) GetNameByCode(ctx context.Context, mappingType model.MappingType, code int32) (string, bool, error) {
	key := fmt.Sprintf("mapping:name:%d:%d", mappingType, code)
	val, err := c.cache.GetCtx(ctx, key)
	if err != nil {
		return "", false, err
	}
	if val == "" {
		return "", false, nil
	}
	return val, true, nil
}

func (c *MappingCache) SetCodeByKey(ctx context.Context, mappingType model.MappingType, name string, code int32, ttl time.Duration) error {
	key := fmt.Sprintf("mapping:code:%d:%s", mappingType, name)
	return c.cache.SetexCtx(ctx, key, strconv.FormatInt(int64(code), 10), int(ttl.Seconds()))
}

func (c *MappingCache) SetNameByKey(ctx context.Context, mappingType model.MappingType, code int32, name string, ttl time.Duration) error {
	key := fmt.Sprintf("mapping:name:%d:%d", mappingType, code)
	return c.cache.SetexCtx(ctx, key, name, int(ttl.Seconds()))
}

func (c *MappingCache) Invalidate(ctx context.Context, mappingType model.MappingType) error {
	listKey := fmt.Sprintf("mapping:list:%d", mappingType)
	_, err := c.cache.DelCtx(ctx, listKey)
	return err
}
