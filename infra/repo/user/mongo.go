package user

import (
	"context"
	"errors"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ IMongoRepo = (*MongoRepo)(nil)

const (
	CollectionName = "user"
	IDPrefix       = "meowpick:user:"
	OpenIDPrefix   = "meowpick:user_openid:"
)

type IMongoRepo interface {
	Insert(ctx context.Context, user *User) (err error)
	Update(ctx context.Context, user *User) (err error)
	FindById(ctx context.Context, userId string) (user *User, err error)
	FindByWXOpenId(ctx context.Context, wxOpenId string) (user *User, err error)
	IsAdmin(ctx context.Context, userId string) (isAdmin bool, err error)
}

type MongoRepo struct {
	conn *monc.Model
}

func NewMongoRepo(config *config.Config) *MongoRepo {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, CollectionName, config.Cache)
	return &MongoRepo{
		conn: conn,
	}
}

func (m *MongoRepo) IsAdmin(ctx context.Context, userId string) (bool, error) {
	user, err := m.FindById(ctx, userId)
	if user == nil {
		return false, err
	}
	return user.Admin, nil
}

func (m *MongoRepo) Insert(ctx context.Context, user *User) error {
	if user.ID == "" {
		user.ID = primitive.NewObjectID().Hex()
		user.CreatedAt = time.Now()
		user.UpdatedAt = user.CreatedAt
		// Username, EmailVerified, Ban, Admin 字段留空 默认为nil/false
	}

	idCacheKey := IDPrefix + user.ID

	if _, err := m.conn.InsertOne(ctx, idCacheKey, user); err != nil {
		return errorx.ErrInsertFailed
	}

	// 单独缓存 openID → _id 映射（如果存在openID）
	if user.OpenId != "" {
		openIDCacheKey := OpenIDPrefix + user.OpenId
		// 仅缓存_id，不是完整用户数据
		if err := m.conn.SetCache(openIDCacheKey, user.ID); err != nil {
			log.Error("ID-OpenID映射缓存失败")
		}
	}

	return nil
}

func (m *MongoRepo) Update(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()

	if user.ID == "" {
		return errorx.ErrInvalidObjectID
	}

	if _, err := m.conn.Collection.UpdateByID(ctx, user.ID, bson.M{"$set": user}); err != nil {
		return errorx.ErrUpdateFailed
	}

	return nil

}

func (m *MongoRepo) FindById(ctx context.Context, userId string) (*User, error) {
	var user *User
	var cacheKey = IDPrefix + userId

	if err := m.conn.FindOne(ctx, cacheKey, user, bson.M{consts.ID: userId}); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, errorx.ErrUserNotFound
		}

		return nil, errorx.ErrFindFailed
	}

	return user, nil
}

func (m *MongoRepo) FindByWXOpenId(ctx context.Context, wxOpenId string) (*User, error) {
	var userID string
	openIDCacheKey := OpenIDPrefix + wxOpenId
	if err := m.conn.GetCache(openIDCacheKey, userID); err == nil {
		// 缓存命中：通过_id查完整用户数据
		return m.FindById(ctx, userID)
	}

	// 若缓存未命中 走数据库查询
	var user User
	if err := m.conn.FindOneNoCache(ctx, &user, bson.M{consts.OpenId: wxOpenId}); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, errorx.ErrUserNotFound
		}
		return nil, errorx.ErrFindFailed
	}
	return &user, nil
}
