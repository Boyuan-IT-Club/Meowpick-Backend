package dto

import (
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/teacher"
	"github.com/google/wire"
)

type ITeacherDTO interface {
	// ToTeacherVO 单个Teacher转TeacherVO (DB to VO)
	ToTeacherVO(ctx context.Context, t *teacher.Teacher) (*cmd.TeacherVO, error)
	// ToTeacher 单个TeacherVO转Teacher (VO to DB)
	ToTeacher(ctx context.Context, vo *cmd.TeacherVO) (*teacher.Teacher, error)
	// ToTeacherVOList Teacher数组转TeacherVO数组 (DB Array to VO Array)
	ToTeacherVOList(ctx context.Context, teachers []*teacher.Teacher) ([]*cmd.TeacherVO, error)
	// ToTeacherList TeacherVO数组转Teacher数组 (VO Array to DB Array)
	ToTeacherList(ctx context.Context, vos []*cmd.TeacherVO) ([]*teacher.Teacher, error)
}

type TeacherDTO struct {
	StaticData *consts.StaticData
}

var TeacherDTOSet = wire.NewSet(
	wire.Struct(new(TeacherDTO), "*"),
	wire.Bind(new(ITeacherDTO), new(*TeacherDTO)),
)

// 单个Teacher转TeacherVO (DB to VO)
func (d *TeacherDTO) ToTeacherVO(ctx context.Context, t *teacher.Teacher) (*cmd.TeacherVO, error) {
	if t == nil {
		return nil, nil
	}

	return &cmd.TeacherVO{
		ID:         t.ID,
		Name:       t.Name,
		Title:      t.Title,
		Department: d.StaticData.GetDepartmentNameByID(t.Department),
	}, nil
}

// 单个TeacherVO转Teacher (VO to DB)
func (d *TeacherDTO) ToTeacher(ctx context.Context, vo *cmd.TeacherVO) (*teacher.Teacher, error) {
	if vo == nil {
		return nil, nil
	}

	return &teacher.Teacher{
		ID:         vo.ID,
		Name:       vo.Name,
		Title:      vo.Title,
		Department: d.StaticData.GetDepartmentIDByName(vo.Department),
	}, nil
}

// Teacher数组转TeacherVO数组 (DB Array to VO Array)
func (d *TeacherDTO) ToTeacherVOList(ctx context.Context, teachers []*teacher.Teacher) ([]*cmd.TeacherVO, error) {
	if len(teachers) == 0 {
		return []*cmd.TeacherVO{}, nil
	}

	teacherVOs := make([]*cmd.TeacherVO, 0, len(teachers))

	for _, t := range teachers {
		teacherVO, err := d.ToTeacherVO(ctx, t)
		if err != nil {
			return nil, err
		}
		if teacherVO != nil {
			teacherVOs = append(teacherVOs, teacherVO)
		}
	}

	return teacherVOs, nil
}

// TeacherVO数组转Teacher数组 (VO Array to DB Array)
func (d *TeacherDTO) ToTeacherList(ctx context.Context, vos []*cmd.TeacherVO) ([]*teacher.Teacher, error) {
	if len(vos) == 0 {
		return []*teacher.Teacher{}, nil
	}

	teachers := make([]*teacher.Teacher, 0, len(vos))

	for _, vo := range vos {
		dbTeacher, err := d.ToTeacher(ctx, vo)
		if err != nil {
			return nil, err
		}
		if dbTeacher != nil {
			teachers = append(teachers, dbTeacher)
		}
	}

	return teachers, nil
}
