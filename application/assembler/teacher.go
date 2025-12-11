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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/google/wire"
)

var _ ITeacherAssembler = (*TeacherAssembler)(nil)

type ITeacherAssembler interface {
	ToTeacherVO(ctx context.Context, t *model.Teacher) (*dto.TeacherVO, error)
	ToTeacher(ctx context.Context, vo *dto.TeacherVO) (*model.Teacher, error)
	ToTeacherVOList(ctx context.Context, teachers []*model.Teacher) ([]*dto.TeacherVO, error)
	ToTeacherList(ctx context.Context, vos []*dto.TeacherVO) ([]*model.Teacher, error)
}

type TeacherAssembler struct {
}

var TeacherAssemblerSet = wire.NewSet(
	wire.Struct(new(TeacherAssembler), "*"),
	wire.Bind(new(ITeacherAssembler), new(*TeacherAssembler)),
)

// ToTeacherVO 单个Teacher转TeacherVO (DB to VO)
func (a *TeacherAssembler) ToTeacherVO(ctx context.Context, t *model.Teacher) (*dto.TeacherVO, error) {
	if t == nil {
		return nil, nil
	}

	return &dto.TeacherVO{
		ID:         t.ID,
		Name:       t.Name,
		Title:      t.Title,
		Department: mapping.Data.GetDepartmentNameByID(t.Department),
	}, nil
}

// ToTeacher 单个TeacherVO转Teacher (VO to DB)
func (a *TeacherAssembler) ToTeacher(ctx context.Context, vo *dto.TeacherVO) (*model.Teacher, error) {
	if vo == nil {
		return nil, nil
	}

	return &model.Teacher{
		ID:         vo.ID,
		Name:       vo.Name,
		Title:      vo.Title,
		Department: mapping.Data.GetDepartmentIDByName(vo.Department),
	}, nil
}

// ToTeacherVOList Teacher数组转TeacherVO数组 (DB Array to VO Array)
func (a *TeacherAssembler) ToTeacherVOList(ctx context.Context, teachers []*model.Teacher) ([]*dto.TeacherVO, error) {
	if len(teachers) == 0 {
		return []*dto.TeacherVO{}, nil
	}

	teacherVOs := make([]*dto.TeacherVO, 0, len(teachers))

	for _, t := range teachers {
		teacherVO, err := a.ToTeacherVO(ctx, t)
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
func (a *TeacherAssembler) ToTeacherList(ctx context.Context, vos []*dto.TeacherVO) ([]*model.Teacher, error) {
	if len(vos) == 0 {
		return []*model.Teacher{}, nil
	}

	teachers := make([]*model.Teacher, 0, len(vos))

	for _, vo := range vos {
		dbTeacher, err := a.ToTeacher(ctx, vo)
		if err != nil {
			return nil, err
		}
		if dbTeacher != nil {
			teachers = append(teachers, dbTeacher)
		}
	}

	return teachers, nil
}
