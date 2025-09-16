package teacher

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	CollectionName = "teachers"
)

type IMongoMapper interface {
	GetTeacherSuggestions(ctx context.Context, req *cmd.GetSearchSuggestReq) ([]*Teacher, int64, error)
	CountTeachers(ctx context.Context, req *cmd.GetSearchSuggestReq) (int64, error)
}

type MongoMapper struct {
	conn *monc.Model
}

func NewMongoMapper(cfg *config.Config) *MongoMapper {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CollectionName, cfg.Cache)
	return &MongoMapper{conn: conn}
}

func (m *MongoMapper) GetTeacherSuggestions(ctx context.Context, req *cmd.GetSearchSuggestReq) ([]*Teacher, error) {
	var teachers []*Teacher
	filter := bson.M{"name": bson.M{"$regex": primitive.Regex{Pattern: req.Keyword, Options: "i"}}}
	pageParam := cmd.PageParam{
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	findOption := util.FindPageOption(&pageParam)

	err := m.conn.Find(ctx, &teachers, filter, findOption)
	if err != nil {
		return nil, err
	}
	return teachers, nil
}

func (m *MongoMapper) CountTeachers(ctx context.Context, req *cmd.GetSearchSuggestReq) (int64, error) {
	filter := bson.M{"name": bson.M{"$regex": primitive.Regex{Pattern: req.Keyword, Options: "i"}}}
	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return total, nil
}
