package like

import (
	"context"
	"errors"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

const (
	CollectionName = "like"
)

type IMongoMapper interface {
	// ToggleLike 翻转点赞状态，返回完成后目标的点赞状态
	ToggleLike(ctx context.Context, userID, targetID string, targetType int32) (bool, error)
	// GetLikeStatus 获取一个用户对一个目标的当前点赞状态（是/否点赞）
	GetLikeStatus(ctx context.Context, userID, targetID string, targetType int32) (bool, error)
	// GetLikeCount 获得目标的总点赞数
	GetLikeCount(ctx context.Context, targetID string, targetType int32) (int64, error)
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

func (m *MongoMapper) ToggleLike(ctx context.Context, userID, targetID string, targetType int32) (bool, error) {
	// 查找现有点赞记录，准备更新
	filter := bson.M{
		"userid":   userID,
		"targetId": targetID,
		//"targetType":targetType, // TODO 未来有需要时加上类型过滤器
	}

	var like Like
	var err error

	newActive := true // 新状态
	if err = m.conn.FindOneNoCache(ctx, &like, filter); err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return false, errorx.ErrFindFailed
		}
	}

	if err == nil {
		newActive = !like.Active
	}

	// 更新
	update := bson.M{
		"$set": bson.M{
			"active":    newActive,
			"updatedAt": time.Now(),
		},
		"$setOnInsert": bson.M{
			"createdAt": time.Now(),
			//"targetType": targetType,
		},
	}
	updateOptions := options.Update().SetUpsert(true)

	if _, err := m.conn.UpdateOneNoCache(ctx, filter, update, updateOptions); err != nil {
		return false, errorx.ErrUpdateFailed
	}

	// 返回结果
	return newActive, nil
}

func (m *MongoMapper) GetLikeStatus(ctx context.Context, userID, targetID string, targetType int32) (bool, error) {
	filter := bson.M{
		"userId":   userID,
		"targetId": targetID,
		//"targetType": targetType,
	}

	var like Like
	var err error
	if err = m.conn.FindOneNoCache(ctx, &like, filter); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, errorx.ErrFindFailed
	}

	return like.Active, nil
}

func (m *MongoMapper) GetLikeCount(ctx context.Context, targetID string, targetType int32) (int64, error) {
	filter := bson.M{
		"targetId": targetID,
		//"targetType": targetType,
	}

	if count, err := m.conn.CountDocuments(ctx, filter); err == nil {
		return count, nil
	}

	return 0, errorx.ErrCountFailed
}
