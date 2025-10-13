package course

import (
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	CacheKeyPrefix = "meowpick:course:"
	CollectionName = "course"
)

type IMongoMapper interface {
	FindOneByID(ctx context.Context, ID string) (*Course, error)
	FindMany(ctx context.Context, keyword string, param *cmd.PageParam) ([]*Course, int64, error)
	GetDepartments(ctx context.Context, keyword string) ([]int32, error)
	GetCategories(ctx context.Context, keyword string) ([]int32, error)
	GetCampuses(ctx context.Context, keyword string) ([]int32, error)
	GetCourseSuggestions(ctx context.Context, keyword string, param *cmd.PageParam) ([]*Course, error)
	CountCourses(ctx context.Context, keyword string) (int64, error)
	FindCoursesByTeacherID(ctx context.Context, teacherID string, param *cmd.PageParam) ([]*Course, int64, error)
	FindCoursesByCategoryID(ctx context.Context, categoryID int32, param *cmd.PageParam) ([]*Course, int64, error)
	FindCoursesByDepartmentID(ctx context.Context, departmentID int32, param *cmd.PageParam) ([]*Course, int64, error)
}

type MongoMapper struct {
	conn *monc.Model
}

func NewMongoMapper(cfg *config.Config) *MongoMapper {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CollectionName, cfg.Cache)
	return &MongoMapper{conn: conn}
}

func (m *MongoMapper) FindOneByID(ctx context.Context, ID string) (*Course, error) {
	// 数据库直接用string存储 无需转换ObjectiveID
	course := &Course{}
	if err := m.conn.FindOneNoCache(ctx, course, bson.M{consts.ID: ID}); err != nil {
		log.Error("No course found with ID：", ID)
		return nil, err
	}
	return course, nil
}

func (m *MongoMapper) FindMany(ctx context.Context, keyword string, param *cmd.PageParam) ([]*Course, int64, error) {

	if keyword == "" {
		return []*Course{}, 0, nil
	}

	//构建查询过滤器 (Filter)
	filter := bson.M{}
	filter["$or"] = []bson.M{{consts.Name: keyword}, {consts.Code: keyword}}

	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*Course{}, 0, nil
	}

	//构建分页和排序选项
	ops := util.FindPageOption(param).SetSort(util.DSort(consts.CreatedAt, -1))

	var courses []*Course //
	err = m.conn.Find(ctx, &courses, filter, ops)
	if err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}

func (m *MongoMapper) GetDepartments(ctx context.Context, keyword string) ([]int32, error) {

	if keyword == "" {
		return nil, nil
	}

	filter := bson.M{"$or": []bson.M{{consts.Name: keyword}, {consts.Code: keyword}}}

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

func (m *MongoMapper) GetCategories(ctx context.Context, keyword string) ([]int32, error) {
	if keyword == "" {
		return nil, nil
	}
	filter := bson.M{"$or": []bson.M{{consts.Name: keyword}, {consts.Code: keyword}}}

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

func (m *MongoMapper) GetCampuses(ctx context.Context, keyword string) ([]int32, error) {
	if keyword == "" {
		return nil, nil
	}
	filter := bson.M{"$or": []bson.M{{consts.Name: keyword}, {consts.Code: keyword}}}
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

func (m *MongoMapper) GetCourseSuggestions(ctx context.Context, keyword string, param *cmd.PageParam) ([]*Course, error) {
	if keyword == "" {
		return nil, nil
	}
	var courses []*Course
	filter := bson.M{consts.Name: bson.M{"$regex": primitive.Regex{Pattern: keyword, Options: "i"}}}
	ops := util.FindPageOption(param)

	err := m.conn.Find(ctx, &courses, filter, ops)
	if err != nil {
		return nil, err
	}

	return courses, nil
}

func (m *MongoMapper) CountCourses(ctx context.Context, keyword string) (int64, error) {
	filter := bson.M{consts.Name: bson.M{"$regex": primitive.Regex{Pattern: keyword, Options: "i"}}}

	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		log.Error("Count All Courses Failed:", err)
		return 0, err
	}
	return total, nil
}

// FindCoursesByTeacherID 根据教师ID查询其教授的所有课程
func (m *MongoMapper) FindCoursesByTeacherID(ctx context.Context, teacherID string, param *cmd.PageParam) ([]*Course, int64, error) {
	if teacherID == "" {
		return nil, 0, errorx.ErrEmptyTeacherID
	}

	var courses []*Course

	// 在 MongoDB 中，对数组字段进行简单的相等查询，会自动查找数组中包含该元素的文档
	filter := bson.M{consts.TeacherIds: teacherID}

	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*Course{}, 0, nil
	}

	ops := util.FindPageOption(param).SetSort(util.DSort(consts.CreatedAt, -1))

	err = m.conn.Find(ctx, &courses, filter, ops)
	if err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}

// FindCoursesByCategoryID 根据课程分类/部门分页查询课程
func (m *MongoMapper) FindCoursesByCategoryID(ctx context.Context, categoryID int32, param *cmd.PageParam) ([]*Course, int64, error) {
	var courses []*Course
	filter := bson.M{consts.Category: categoryID}

	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*Course{}, 0, nil
	}

	ops := util.FindPageOption(param).SetSort(util.DSort(consts.CreatedAt, -1))

	err = m.conn.Find(ctx, &courses, filter, ops)
	if err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}

func (m *MongoMapper) FindCoursesByDepartmentID(ctx context.Context, departmentID int32, param *cmd.PageParam) ([]*Course, int64, error) {
	var courses []*Course
	filter := bson.M{consts.Department: departmentID}

	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*Course{}, 0, nil
	}

	ops := util.FindPageOption(param).SetSort(util.DSort(consts.CreatedAt, -1))

	err = m.conn.Find(ctx, &courses, filter, ops)
	if err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}
