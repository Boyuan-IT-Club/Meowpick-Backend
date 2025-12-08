// Copyright 2025 Boyuan-IT-Club
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package assembler

import (
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/teacher"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/google/wire"
)

var _ ITeacherDTO = (*TeacherDTO)(nil)

type ITeacherDTO interface {
	ToTeacherVO(ctx context.Context, t *teacher.Teacher) (*dto.TeacherVO, error)
	ToTeacher(ctx context.Context, vo *dto.TeacherVO) (*teacher.Teacher, error)
	ToTeacherVOList(ctx context.Context, teachers []*teacher.Teacher) ([]*dto.TeacherVO, error)
	ToTeacherList(ctx context.Context, vos []*dto.TeacherVO) ([]*teacher.Teacher, error)
}

type TeacherDTO struct {
	StaticData *mapping.StaticData
}

var TeacherDTOSet = wire.NewSet(
	wire.Struct(new(TeacherDTO), "*"),
	wire.Bind(new(ITeacherDTO), new(*TeacherDTO)),
)

// ToTeacherVO 单个Teacher转TeacherVO (DB to VO)
func (d *TeacherDTO) ToTeacherVO(ctx context.Context, t *teacher.Teacher) (*dto.TeacherVO, error) {
	if t == nil {
		return nil, nil
	}

	return &dto.TeacherVO{
		ID:         t.ID,
		Name:       t.Name,
		Title:      t.Title,
		Department: d.StaticData.GetDepartmentNameByID(t.Department),
	}, nil
}

// ToTeacher 单个TeacherVO转Teacher (VO to DB)
func (d *TeacherDTO) ToTeacher(ctx context.Context, vo *dto.TeacherVO) (*teacher.Teacher, error) {
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

// ToTeacherVOList Teacher数组转TeacherVO数组 (DB Array to VO Array)
func (d *TeacherDTO) ToTeacherVOList(ctx context.Context, teachers []*teacher.Teacher) ([]*dto.TeacherVO, error) {
	if len(teachers) == 0 {
		return []*dto.TeacherVO{}, nil
	}

	teacherVOs := make([]*dto.TeacherVO, 0, len(teachers))

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

// ToTeacherList TeacherVO数组转Teacher数组 (VO Array to DB Array)
func (d *TeacherDTO) ToTeacherList(ctx context.Context, vos []*dto.TeacherVO) ([]*teacher.Teacher, error) {
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
