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
	UserCollectionName = "user"
	UserIDPrefix       = "meowpick:user:"
	UserOpenIDPrefix   = "meowpick:user_openid:"
)

type IUserRepo interface {
	Insert(ctx context.Context, user *model.User) (err error)
	Update(ctx context.Context, user *model.User) (err error)

	FindByID(ctx context.Context, userId string) (user *model.User, err error)
	FindByOpenID(ctx context.Context, openId string) (user *model.User, err error)

	IsAdminByID(ctx context.Context, userId string) (isAdmin bool, err error)
}

type UserRepo struct {
	conn *monc.Model
}

func NewUserRepo(config *config.Config) *UserRepo {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, UserCollectionName, config.Cache)
	return &UserRepo{conn: conn}
}

// Insert 插入用户
func (r *UserRepo) Insert(ctx context.Context, user *model.User) error {
	if _, err := r.conn.InsertOne(ctx, UserIDPrefix+user.ID, user); err != nil {
		return err
	}
	// 单独缓存 openID → _id 映射（如果存在openID）
	if user.OpenId != "" {
		openIDCacheKey := UserOpenIDPrefix + user.OpenId
		// 仅缓存_id，不是完整用户数据
		if err := r.conn.SetCache(openIDCacheKey, user.ID); err != nil {
			logs.Warnf("failed to cache openId to userId mapping: %v", err)
		}
	}
	return nil
}

// Update 更新用户信息
func (r *UserRepo) Update(ctx context.Context, user *model.User) error {
	if _, err := r.conn.Collection.UpdateByID(ctx, user.ID, bson.M{"$set": user}); err != nil {
		return err
	}
	return nil
}

// FindByID 通过ID查询用户
func (r *UserRepo) FindByID(ctx context.Context, userId string) (*model.User, error) {
	user := &model.User{}
	if err := r.conn.FindOne(ctx, UserIDPrefix+userId, user, bson.M{consts.ID: userId}); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

// FindByOpenID 通过OpenID查询用户
func (r *UserRepo) FindByOpenID(ctx context.Context, openId string) (*model.User, error) {
	var userID string
	// 缓存命中 通过_id查完整用户数据
	if err := r.conn.GetCache(UserOpenIDPrefix+openId, userID); err == nil {
		return r.FindByID(ctx, userID)
	}
	// 若缓存未命中 走数据库查询
	var user model.User
	if err := r.conn.FindOneNoCache(ctx, &user, bson.M{consts.OpenID: openId}); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// IsAdminByID 判断用户是否是管理员
func (r *UserRepo) IsAdminByID(ctx context.Context, userId string) (bool, error) {
	user, err := r.FindByID(ctx, userId)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, monc.ErrNotFound
	}
	return user.Admin, nil
}
