package repo

import (
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
)

var _ IProposalRepo = (*ProposalRepo)(nil)

const (
	ProposalCollectionName = "proposal"
)

type IProposalRepo interface {
	Insert(ctx context.Context, proposal *model.Proposal) error
	IsCourseInExistingProposals(ctx context.Context, courseVO *dto.CourseVO) (bool, error)
}

type ProposalRepo struct {
	conn *monc.Model
}

func NewProposalRepo(cfg *config.Config) *ProposalRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, ProposalCollectionName, cfg.Cache)
	return &ProposalRepo{conn: conn}
}

// Insert 插入一个新的提案
func (r *ProposalRepo) Insert(ctx context.Context, proposal *model.Proposal) error {
	_, err := r.conn.InsertOneNoCache(ctx, proposal)
	return err
}

// IsCourseInExistingProposals 检查课程是否已经存在于现有提案中
// 比较的字段包括: Name, Code, Department, Category, Campuses, TeacherIDs
func (s *ProposalRepo) IsCourseInExistingProposals(ctx context.Context, courseVO *dto.CourseVO) (bool, error) {
	// 将DTO中的值转换为ID形式以便数据库查询
	departmentID := mapping.Data.GetDepartmentIDByName(courseVO.Department)
	categoryID := mapping.Data.GetCategoryIDByName(courseVO.Category)

	// 将校区名称转换为ID
	campusIDs := make([]int32, len(courseVO.Campuses))
	for i, campus := range courseVO.Campuses {
		campusIDs[i] = mapping.Data.GetCampusIDByName(campus)
	}

	// 构造查询条件，检查提案中的课程字段
	filter := bson.M{
		"course.name":       courseVO.Name,
		"course.code":       courseVO.Code,
		"course.department": departmentID,
		"course.category":   categoryID,
		"course.campuses":   bson.M{"$all": campusIDs, "$size": len(campusIDs)},
		"deleted":           false, // 只检查未删除的提案
	}

	// 如果提供了教师信息，则也加入查询条件
	if len(courseVO.Teachers) > 0 {
		teacherIDs := make([]string, len(courseVO.Teachers))
		for i, teacher := range courseVO.Teachers {
			teacherIDs[i] = teacher.ID
		}
		filter["course.teacherIds"] = bson.M{"$all": teacherIDs, "$size": len(teacherIDs)}
	} else {
		// 如果没有提供教师信息，则查询teacherIds为空或者不存在的记录
		filter["$or"] = []bson.M{
			{"course.teacherIds": bson.M{"$exists": false}},
			{"course.teacherIds": bson.M{"$size": 0}},
		}
	}

	// 查询提案中是否已存在该课程
	count, err := s.conn.CountDocuments(ctx, filter)
	if err != nil {
		return false, errorx.WrapByCode(err, errno.ErrProposalCourseFindInProposalFailed,
			errorx.KV("operation", "check proposal course existence"))
	}

	return count > 0, nil
}
