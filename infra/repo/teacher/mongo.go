package teacher

import (
	"context"
	"errors"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ IMongoRepo = (*MongoRepo)(nil)

const (
	CacheKeyPrefix  = "meowpick:teacher:"
	CollectionName  = "teacher"
	Name2IDCacheKey = "meowpick:teacher_name_to_id:" // 再建一套name->ID的缓存
)

type IMongoRepo interface {
	AddNewTeacher(ctx context.Context, teacher *Teacher) (ID string, err error)
	FindOneTeacherByID(ctx context.Context, ID string) (*Teacher, error)
	GetTeacherSuggestions(ctx context.Context, keyword string, param *dto.PageParam) ([]*Teacher, error)
	CountTeachers(ctx context.Context, keyword string) (int64, error)
	GetTeacherIDByName(ctx context.Context, name string) (string, error)
}

type MongoRepo struct {
	conn *monc.Model
}

func NewMongoRepo(cfg *config.Config) *MongoRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CollectionName, cfg.Cache)
	return &MongoRepo{conn: conn}
}

func (m *MongoRepo) AddNewTeacher(ctx context.Context, teacher *Teacher) (ID string, err error) {
	cacheKey := CacheKeyPrefix + teacher.ID

	_, err = m.conn.InsertOne(ctx, cacheKey, teacher)
	if err != nil {
		return "", errorx.ErrInsertFailed
	}

	// 设置name->id映射缓存
	nameCacheKey := Name2IDCacheKey + teacher.Name
	if err = m.conn.SetCache(nameCacheKey, cacheKey); err != nil {
		log.Error("Failed to set name-to-id mapping cache:", err)
		// 不返回错误，继续执行
	}

	return teacher.ID, nil
}

func (m *MongoRepo) FindOneTeacherByID(ctx context.Context, ID string) (*Teacher, error) {
	var teacher Teacher
	//cacheKey := CacheKeyPrefix + ID
	err := m.conn.FindOneNoCache(ctx, &teacher, bson.M{consts.ID: ID})
	if err != nil {
		return nil, err
	}

	return &teacher, nil
}

func (m *MongoRepo) GetTeacherSuggestions(ctx context.Context, keyword string, param *dto.PageParam) ([]*Teacher, error) {
	var teachers []*Teacher
	filter := bson.M{consts.Name: bson.M{"$regex": primitive.Regex{Pattern: keyword, Options: "i"}}}
	ops := util.FindPageOption(param)

	err := m.conn.Find(ctx, &teachers, filter, ops)
	if err != nil {
		return nil, err
	}
	return teachers, nil
}

func (m *MongoRepo) CountTeachers(ctx context.Context, keyword string) (int64, error) {
	filter := bson.M{consts.Name: bson.M{"$regex": primitive.Regex{Pattern: keyword, Options: "i"}}}
	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (m *MongoRepo) GetTeacherIDByName(ctx context.Context, name string) (string, error) {
	nameCacheKey := Name2IDCacheKey + name
	var teacherID string
	// 先查name-id缓存
	if err := m.conn.GetCache(nameCacheKey, &teacherID); err != nil && teacherID != "" {
		return teacherID, nil
	}
	filter := bson.M{consts.Name: name}
	var teacher Teacher

	// 使用NoCache版本，避免使用错误的缓存键
	if err := m.conn.FindOneNoCache(ctx, &teacher, filter); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return "", errorx.ErrFindSuccessButNoResult
		}
		return "", errorx.ErrFindFailed
	}
	// 设置缓存键
	cacheKey := CacheKeyPrefix + teacher.ID
	if err := m.conn.SetCache(nameCacheKey, cacheKey); err != nil {
		log.Error("Failed to set name-to-id mapping cache for teacher:", nameCacheKey, err)
	}

	return teacher.ID, nil
}
