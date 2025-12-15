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

package repo

import (
	"context"
	"errors"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
)

var _ IUserRepo = (*UserRepo)(nil)

const (
	UserCollectionName   = "user"
	UserOpenID2UserIDKey = consts.CacheUserKeyPrefix + "openId2id:"
	UserID2DBKey         = consts.CacheUserKeyPrefix + "id2db:"
)

type IUserRepo interface {
	Insert(ctx context.Context, user *model.User) (err error)
	Update(ctx context.Context, user *model.User) (err error)

	FindByID(ctx context.Context, id string) (user *model.User, err error)
	FindByOpenID(ctx context.Context, openId string) (user *model.User, err error)

	IsAdminByID(ctx context.Context, id string) (isAdmin bool, err error)
}

type UserRepo struct {
	conn *monc.Model
}

func NewUserRepo(cfg *config.Config) *UserRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, UserCollectionName, cfg.Cache)
	return &UserRepo{conn: conn}
}

// Insert 插入用户
func (r *UserRepo) Insert(ctx context.Context, user *model.User) error {
	if _, err := r.conn.InsertOne(ctx, UserID2DBKey+user.ID, user); err != nil {
		return err
	}
	// 单独缓存 openId → userId 映射（如果存在openId）
	if user.OpenID != "" {
		if err := r.conn.SetCache(UserOpenID2UserIDKey+user.OpenID, user.ID); err != nil {
			logs.CtxWarnf(ctx, "[monc] [SetCache] set openId to userId cache error: %v", err)
		}
	}
	return nil
}

// Update 更新用户信息
func (r *UserRepo) Update(ctx context.Context, user *model.User) error {
	if _, err := r.conn.UpdateOne(ctx, UserID2DBKey+user.ID,
		bson.M{consts.ID: user.ID}, bson.M{"$set": user}); err != nil {
		return err
	}
	return nil
}

// FindByID 通过ID查询用户
func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	user := &model.User{}
	if err := r.conn.FindOne(ctx, UserID2DBKey+id, user, bson.M{consts.ID: id}); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

// FindByOpenID 通过OpenID查询用户
func (r *UserRepo) FindByOpenID(ctx context.Context, openId string) (*model.User, error) {
	var userId string
	// 缓存命中 通过_id查完整用户数据
	if err := r.conn.GetCache(UserOpenID2UserIDKey+openId, &userId); err == nil && userId != "" {
		return r.FindByID(ctx, userId)
	}
	// 若缓存未命中 走数据库查询
	user := model.User{}
	if err := r.conn.FindOneNoCache(ctx, &user, bson.M{consts.OpenID: openId}); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	// 回写缓存
	if err := r.conn.SetCache(UserOpenID2UserIDKey+openId, user.ID); err != nil {
		logs.CtxWarnf(ctx, "[monc] [SetCache] set openId to userId cache error: %v", err)
	}
	return &user, nil
}

// IsAdminByID 判断用户是否是管理员
func (r *UserRepo) IsAdminByID(ctx context.Context, id string) (bool, error) {
	user, err := r.FindByID(ctx, id)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, monc.ErrNotFound
	}
	return user.Admin, nil
}
