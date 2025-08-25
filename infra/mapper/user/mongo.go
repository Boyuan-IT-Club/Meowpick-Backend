package user

import (
	"context"
	"errors"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

const (
	CollectionName = "user"
	IDPrefix       = "user:"
	OpenIDPrefix   = "user_openid:"
)

type IMongoMapper interface {
	Insert(ctx context.Context, user *User) (err error)
	Update(ctx context.Context, user *User) (err error)

	FindById(ctx context.Context, userId string) (user *User, err error)
	FindByWXOpenId(ctx context.Context, wxOpenId string) (user *User, err error)
}

type MongoMapper struct {
	conn *monc.Model
}

func NewMongoMapper(config *config.Config) *MongoMapper {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, CollectionName, config.Cache)
	return &MongoMapper{
		conn: conn,
	}
}

func (m *MongoMapper) Insert(ctx context.Context, user *User) error {
	if user.ID.IsZero() {
		user.ID = primitive.NewObjectID()
		user.CreatedAt = time.Now()
		user.UpdatedAt = user.CreatedAt
		// Username, EmailVerified, Ban, Admin 字段留空 默认为nil/false
	}

	idCacheKey := IDPrefix + user.ID.Hex()

	if _, err := m.conn.InsertOne(ctx, idCacheKey, user); err != nil {
		return errorx.ErrInsertFailed
	}

	// 单独缓存 openID → _id 映射（如果存在openID）
	if user.OpenId != "" {
		openIDCacheKey := OpenIDPrefix + user.OpenId
		// 仅缓存_id，不是完整用户数据
		if err := m.conn.SetCache(openIDCacheKey, user.ID.Hex()); err != nil {
			log.Error("ID-OpenID映射缓存失败")
		}
	}

	return nil
}

func (m *MongoMapper) Update(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()

	if user.ID.IsZero() {
		return errorx.ErrInvalidObjectID
	}

	if _, err := m.conn.Collection.UpdateByID(ctx, user.ID, bson.M{"$set": user}); err != nil {
		return errorx.ErrUpdateFailed
	}

	return nil

}

func (m *MongoMapper) FindById(ctx context.Context, userId string) (*User, error) {
	var user *User
	var cacheKey = IDPrefix + userId

	if err := m.conn.FindOne(ctx, cacheKey, user, bson.M{"_id": userId}); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, errorx.ErrUserNotFound
		}

		return nil, errorx.ErrFindFailed
	}

	return user, nil
}

func (m *MongoMapper) FindByWXOpenId(ctx context.Context, wxOpenId string) (*User, error) {
	var userID string
	openIDCacheKey := OpenIDPrefix + wxOpenId
	if err := m.conn.GetCache(openIDCacheKey, userID); err == nil {
		// 缓存命中：通过_id查完整用户数据
		return m.FindById(ctx, userID)
	}

	// 若缓存未命中 走数据库查询
	var user User
	if err := m.conn.FindOneNoCache(ctx, &user, bson.M{"openId": wxOpenId}); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, errorx.ErrUserNotFound
		}
		return nil, errorx.ErrFindFailed
	}
	return &user, nil
}
