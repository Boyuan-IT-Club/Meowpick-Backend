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
	"fmt"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ IUserRepo = (*UserRepo)(nil)

const (
	UserCollectionName    = "user"
	UserOpenID2UserIDKey  = consts.CacheUserKeyPrefix + "openId2id:"
	UserID2DBKey          = consts.CacheUserKeyPrefix + "id2db:"
	UserUsernameIndexName = "idx_user_username_unique"
)

type IUserRepo interface {
	Insert(ctx context.Context, user *model.User) (err error)
	Update(ctx context.Context, user *model.User) (err error)

	FindByID(ctx context.Context, id string) (user *model.User, err error)
	FindByIDs(ctx context.Context, ids []string) (users []*model.User, err error)
	FindByOpenID(ctx context.Context, openId string) (user *model.User, err error)
	IsUsernameExist(ctx context.Context, username, excludeUserID string) (bool, error)
	UpdateProfile(ctx context.Context, id string, username, avatar *string, usernameUpdatedAt, expectedUsernameUpdatedAt *time.Time) (bool, error)
	SetAdmin(ctx context.Context, id string, admin bool) error
	InvalidateByID(ctx context.Context, id string) error

	IsAdminByID(ctx context.Context, id string) (isAdmin bool, err error)
	IncrementContribution(ctx context.Context, id string, delta int64) error
}

type UserRepo struct {
	conn *monc.Model
}

func NewUserRepo(cfg *config.Config) (*UserRepo, error) {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, UserCollectionName, cfg.Cache)
	repository := &UserRepo{conn: conn}
	if err := repository.ensureIndexes(context.Background()); err != nil {
		return nil, err
	}
	return repository, nil
}

// ensureIndexes 为非空昵称创建忽略大小写的唯一索引。
func (r *UserRepo) ensureIndexes(ctx context.Context) error {
	_, err := r.conn.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: consts.Username, Value: 1}},
		Options: options.Index().
			SetName(UserUsernameIndexName).
			SetUnique(true).
			SetCollation(usernameCollation()).
			SetPartialFilterExpression(bson.M{
				consts.Username: bson.M{"$type": "string", "$gt": ""},
			}),
	})
	return err
}

func usernameCollation() *options.Collation {
	return &options.Collation{Locale: "en", Strength: 2}
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
	var err error
	if _, inTransaction := ctx.(mongo.SessionContext); inTransaction {
		err = r.conn.FindOneNoCache(ctx, user, bson.M{consts.ID: id})
	} else {
		err = r.conn.FindOne(ctx, UserID2DBKey+id, user, bson.M{consts.ID: id})
	}
	if err != nil {
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

// IsUsernameExist 检查忽略大小写后的昵称是否已被其他用户占用。
func (r *UserRepo) IsUsernameExist(ctx context.Context, username, excludeUserID string) (bool, error) {
	filter := bson.M{consts.Username: username}
	if excludeUserID != "" {
		filter[consts.ID] = bson.M{"$ne": excludeUserID}
	}

	var user model.User
	err := r.conn.FindOneNoCache(ctx, &user, filter,
		options.FindOne().SetProjection(bson.M{consts.ID: 1}).SetCollation(usernameCollation()),
	)
	if errors.Is(err, monc.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// UpdateProfile 原子更新用户资料中实际传入且发生变化的字段。
func (r *UserRepo) UpdateProfile(
	ctx context.Context,
	id string,
	username, avatar *string,
	usernameUpdatedAt, expectedUsernameUpdatedAt *time.Time,
) (bool, error) {
	set := bson.M{consts.UpdatedAt: time.Now()}
	if username != nil {
		set[consts.Username] = *username
	}
	if avatar != nil {
		set[consts.Avatar] = *avatar
	}
	if usernameUpdatedAt != nil {
		set[consts.UsernameUpdatedAt] = *usernameUpdatedAt
	}

	filter := bson.M{consts.ID: id}
	if expectedUsernameUpdatedAt != nil {
		if expectedUsernameUpdatedAt.IsZero() {
			filter["$or"] = bson.A{
				bson.M{consts.UsernameUpdatedAt: bson.M{"$exists": false}},
				bson.M{consts.UsernameUpdatedAt: time.Time{}},
			}
		} else {
			filter[consts.UsernameUpdatedAt] = *expectedUsernameUpdatedAt
		}
	}

	result, err := r.conn.UpdateOne(ctx, UserID2DBKey+id, filter, bson.M{"$set": set})
	if err != nil {
		return false, err
	}
	return result.MatchedCount == 1, nil
}

// SetAdmin updates only the privilege bit and timestamp. Using a partial User
// struct with Update would overwrite unrelated identity/profile fields with zeros.
func (r *UserRepo) SetAdmin(ctx context.Context, id string, admin bool) error {
	_, err := r.conn.UpdateOne(ctx, UserID2DBKey+id,
		bson.M{consts.ID: id},
		bson.M{"$set": bson.M{"admin": admin, consts.UpdatedAt: time.Now()}},
	)
	return err
}

func (r *UserRepo) InvalidateByID(ctx context.Context, id string) error {
	return r.conn.DelCache(ctx, UserID2DBKey+id)
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

// IncrementContribution 原子增减用户贡献值（delta 可为负，用于撤回时扣减）
func (r *UserRepo) IncrementContribution(ctx context.Context, id string, delta int64) error {
	filter := bson.M{consts.ID: id}
	if delta < 0 {
		filter[consts.UserContribution] = bson.M{"$gte": -delta}
	}
	update := bson.M{
		"$inc": bson.M{consts.UserContribution: delta},
	}
	result, err := r.conn.UpdateOneNoCache(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount != 1 {
		return fmt.Errorf("user %s not found or contribution would become negative", id)
	}
	if _, inTransaction := ctx.(mongo.SessionContext); !inTransaction {
		if err := r.InvalidateByID(ctx, id); err != nil {
			logs.CtxWarnf(ctx, "[monc] [DelCache] delete user cache error: %v", err)
		}
	}
	return nil
}

// FindByIDs 根据用户ID列表批量查询用户
func (r *UserRepo) FindByIDs(ctx context.Context, ids []string) ([]*model.User, error) {
	if len(ids) == 0 {
		return []*model.User{}, nil
	}

	users := []*model.User{}
	filter := bson.M{
		consts.ID: bson.M{"$in": ids},
	}

	if err := r.conn.Find(ctx, &users, filter); err != nil {
		logs.CtxErrorf(ctx, "[UserRepo] [FindByIDs] error: %v", err)
		return nil, err
	}

	return users, nil
}
