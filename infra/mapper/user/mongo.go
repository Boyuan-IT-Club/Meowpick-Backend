package user

import (
	"context"
	"errors"
	"fmt"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

const (
	CollectionName = "user_server"
)

type IMongoMapper interface {
	Insert(ctx context.Context, user *User) (err error)
	Update(ctx context.Context, user *User) (err error)

	FindById(ctx context.Context, userId primitive.ObjectID) (user *User, err error)
	FindByWXOpenId(ctx context.Context, wxOpenId string) (user *User, err error)
}

type MongoMapper struct {
	conn *monc.Model
}

func NewMongoMapper(config config.Config) IMongoMapper {
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
	_, err := m.conn.InsertOneNoCache(ctx, user)
	if err != nil {
		return errorx.ErrUserInsertFailed
	}
	return nil
}

func (m *MongoMapper) Update(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()
	_, err := m.conn.UpdateByIDNoCache(ctx, user.ID, bson.M{"$set": user})
	return err
}

func (m *MongoMapper) FindById(ctx context.Context, userId primitive.ObjectID) (*User, error) {
	var user User
	err := m.conn.FindOneNoCache(ctx, &user, bson.M{"_id": userId})
	if err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, errorx.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user_server by id failed: %w", err)
	}
	return &user, nil
}

func (m *MongoMapper) FindByWXOpenId(ctx context.Context, wxOpenId string) (*User, error) {
	var user User
	err := m.conn.FindOneNoCache(ctx, &user, bson.M{"openId": wxOpenId})
	if err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, errorx.ErrUserNotFound
		}
		return nil, errorx.ErrFindFailed
	}
	return &user, nil
}
