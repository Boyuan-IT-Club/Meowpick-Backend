package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/google/wire"
)

type ITeacherService interface {
	ListCoursesByTeacher(ctx context.Context, req *cmd.GetTeachersReq) (*cmd.GetTeachersResp, error)
}

type TeacherService struct {
	CourseMapper  *course.MongoMapper
	StaticData    *consts.StaticData
	CommentMapper *comment.MongoMapper
}

var TeacherServiceSet = wire.NewSet(
	wire.Struct(new(TeacherService), "*"),
	wire.Bind(new(ITeacherService), new(*TeacherService)),
)

func (s *TeacherService) ListCoursesByTeacher(ctx context.Context, req *cmd.GetTeachersReq) (*cmd.GetTeachersResp, error) {
	courseListFromDB, total, err := s.CourseMapper.FindCoursesByTeacherID(ctx, req)

	if err != nil {
		return nil, err
	}

	// 将从数据库拿到的 course.Course 模型，转换为前端需要的 DTO 模型。
	// Optimize: 抽象出dto层处理数据转换
	var courseDTOList []*cmd.CourseVO
	for _, dbCourse := range courseListFromDB {
		//先处理校区
		campusNames := make([]string, 0, len(dbCourse.Campuses))
		for _, dbCampus := range dbCourse.Campuses {
			campusName := s.StaticData.GetCampusNameByID(dbCampus)
			campusNames = append(campusNames, campusName)
		}
		// 处理tagCount

		tagCount, err := s.CommentMapper.CountCourseTag(ctx, dbCourse.ID)
		if err != nil {
			log.Error("CountCourseTag Failed, courseID: ", dbCourse, err)
			return nil, errorx.ErrCountCourseTagsFailed
		}
		apiCourse := &cmd.CourseVO{
			ID:         dbCourse.ID,
			Name:       dbCourse.Name,
			Code:       dbCourse.Code,
			Department: s.StaticData.GetDepartmentNameByID(dbCourse.Department),
			Category:   s.StaticData.GetCategoryNameByID(dbCourse.Category),
			Campus:     campusNames,
			Teachers:   dbCourse.TeacherIDs,
			TagCount:   tagCount,
		}
		courseDTOList = append(courseDTOList, apiCourse)
	}

	response := &cmd.GetTeachersResp{
		Resp: cmd.Success(),
		PaginatedCourses: &cmd.PaginatedCourses{
			List:  courseDTOList,
			Total: total,
			PageParam: &cmd.PageParam{
				Page:     req.Page,
				PageSize: req.PageSize,
			},
		},
	}

	return response, nil
}
