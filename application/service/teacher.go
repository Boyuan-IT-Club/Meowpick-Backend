package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/google/wire"
)

type ITeacherService interface {
	ListCoursesByTeacher(ctx context.Context, req *cmd.GetTeachersReq) (*cmd.GetTeachersResp, error)
}

type TeacherService struct {
	CourseMapper *course.MongoMapper
	StaticData   *consts.StaticData
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
	var courseDTOList []cmd.CourseInList
	for _, dbCourse := range courseListFromDB {
		//先处理校区
		campusNames := make([]string, 0, len(dbCourse.Campuses))
		for _, dbCampus := range dbCourse.Campuses {
			campusName := s.StaticData.GetCampusNameByID(dbCampus)
			campusNames = append(campusNames, campusName)
		}

		apiCourse := cmd.CourseInList{
			ID:             dbCourse.ID,
			Name:           dbCourse.Name,
			Code:           dbCourse.Code,
			DepartmentName: s.StaticData.GetDepartmentNameByID(dbCourse.Department),
			CategoriesName: s.StaticData.GetCategoryNameByID(dbCourse.Category),
			CampusesName:   campusNames,
			TeachersName:   dbCourse.TeacherIDs,
			// ... 其他需要返回给前端的字段
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
