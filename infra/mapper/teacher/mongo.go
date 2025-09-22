package teacher

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	CollectionName = "teacher"
)

type IMongoMapper interface {
	AddNewTeacher(ctx context.Context, teacherVO *cmd.TeacherVO) (ID string, err error)
	FindOneTeacherByID(ctx context.Context, ID string) (*Teacher, error)
	FindOneTeacherByVO(ctx context.Context, vO *cmd.TeacherVO) (*Teacher, error)
	GetTeacherSuggestions(ctx context.Context, keyword string, param *cmd.PageParam) ([]*Teacher, error)
	CountTeachers(ctx context.Context, keyword string) (int64, error)
	GetTeacherIDByName(ctx context.Context, name string) (string, error)
}

type MongoMapper struct {
	conn *monc.Model
}

func NewMongoMapper(cfg *config.Config) *MongoMapper {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CollectionName, cfg.Cache)
	return &MongoMapper{conn: conn}
}

func (m *MongoMapper) AddNewTeacher(ctx context.Context, teacherVO *cmd.TeacherVO) (ID string, err error) {
	//TODO implement me
	return "", nil
}

func (m *MongoMapper) FindOneTeacherByID(ctx context.Context, ID string) (*Teacher, error) {
	var teacher Teacher
	err := m.conn.FindOneNoCache(ctx, &teacher, bson.M{consts.ID: ID})
	if err != nil {
		return nil, err
	}

	return &teacher, nil
}

func (m *MongoMapper) FindOneTeacherByVO(ctx context.Context, vO *cmd.TeacherVO) (*Teacher, error) {
	//TODO implement me
	return nil, nil
}

func (m *MongoMapper) GetTeacherSuggestions(ctx context.Context, keyword string, param *cmd.PageParam) ([]*Teacher, error) {
	var teachers []*Teacher
	filter := bson.M{consts.Name: bson.M{"$regex": primitive.Regex{Pattern: keyword, Options: "i"}}}
	ops := util.FindPageOption(param)

	err := m.conn.Find(ctx, &teachers, filter, ops)
	if err != nil {
		return nil, err
	}
	return teachers, nil
}

func (m *MongoMapper) CountTeachers(ctx context.Context, keyword string) (int64, error) {
	filter := bson.M{consts.Name: bson.M{"$regex": primitive.Regex{Pattern: keyword, Options: "i"}}}
	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (m *MongoMapper) GetTeacherIDByName(ctx context.Context, name string) (string, error) {
	filter := bson.M{consts.Name: name}
	var teacher Teacher
	if err := m.conn.FindOneNoCache(ctx, &teacher, filter); err != nil {
		return "", errorx.ErrFindFailed
	}
	if teacher.ID == "" {
		return "", errorx.ErrFindSuccessButNoResult
	}
	return teacher.ID, nil
}
