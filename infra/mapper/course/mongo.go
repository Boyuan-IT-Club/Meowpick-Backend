package course

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	CollectionName = "courses"
)

type IMongoMapper interface {
	Find(ctx context.Context, query cmd.GetCoursesReq) ([]*Course, int64, error)
	GetDeparts(ctx context.Context, req cmd.GetCoursesDepartsReq) (*cmd.GetCoursesResp, error)
	GetCategories(ctx context.Context, req *cmd.GetCourseCategoriesReq) ([]int32, error)
	GetCampuses(ctx context.Context, req *cmd.GetCourseCampusesReq) ([]int32, error)
}

type MongoMapper struct {
	conn *monc.Model
}

func NewMongoMapper(cfg *config.Config) *MongoMapper {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CollectionName, cfg.Cache)
	return &MongoMapper{conn: conn}
}

func (m *MongoMapper) Find(ctx context.Context, query *cmd.GetCoursesReq) ([]*Course, int64, error) {

	if query.Keyword == "" {
		return []*Course{}, 0, nil
	}

	//构建查询过滤器 (Filter)
	filter := bson.M{}
	filter["$or"] = []bson.M{
		{"name": query.Keyword},
		{"code": query.Keyword},
	}

	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*Course{}, 0, nil
	}

	//构建分页和排序选项
	findOptions := util.FindPageOption(query).SetSort(util.DSort(consts.CreatedAt, -1))

	var courses []*Course //
	err = m.conn.Find(ctx, &courses, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}

func (m *MongoMapper) GetDeparts(ctx context.Context, req *cmd.GetCoursesDepartsReq) ([]int32, error) {

	if req.Keyword == "" {
		return nil, nil
	}

	filter := bson.M{"$or": []bson.M{{"name": req.Keyword}, {"code": req.Keyword}}}

	results, err := m.conn.Distinct(ctx, consts.Department, filter)
	if err != nil {
		return nil, err
	}

	departmentIDs := make([]int32, 0, len(results))
	for _, result := range results {
		if id, ok := result.(int32); ok {
			departmentIDs = append(departmentIDs, id)
		}
	}

	return departmentIDs, nil
}

func (m *MongoMapper) GetCategories(ctx context.Context, req *cmd.GetCourseCategoriesReq) ([]int32, error) {
	if req.Keyword == "" {
		return nil, nil
	}
	filter := bson.M{"$or": []bson.M{{"name": req.Keyword}, {"code": req.Keyword}}}

	results, err := m.conn.Distinct(ctx, consts.Categories, filter)
	if err != nil {
		return nil, err
	}
	categories := make([]int32, 0, len(results))
	for _, result := range results {
		if id, ok := result.(int32); ok {
			categories = append(categories, id)
		}
	}
	return categories, nil
}

func (m *MongoMapper) GetCampuses(ctx context.Context, req *cmd.GetCourseCampusesReq) ([]int32, error) {
	if req.Keyword == "" {
		return nil, nil
	}
	filter := bson.M{"$or": []bson.M{{"name": req.Keyword}, {"code": req.Keyword}}}
	results, err := m.conn.Distinct(ctx, consts.Campuses, filter)
	if err != nil {
		return nil, err
	}
	campuses := make([]int32, 0, len(results))

	for _, result := range results {
		if id, ok := result.(int32); ok {
			campuses = append(campuses, id)
		}
	}
	return campuses, nil
}
