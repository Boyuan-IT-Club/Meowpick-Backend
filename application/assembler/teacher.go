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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/google/wire"
)

var _ ITeacherAssembler = (*TeacherAssembler)(nil)

type ITeacherAssembler interface {
	ToTeacherVO(db *model.Teacher) *dto.TeacherVO
	ToTeacherDB(vo *dto.TeacherVO) *model.Teacher
	ToTeacherVOArray(dbs []*model.Teacher) []*dto.TeacherVO
	ToTeacherDBArray(vos []*dto.TeacherVO) []*model.Teacher
}

type TeacherAssembler struct {
}

var TeacherAssemblerSet = wire.NewSet(
	wire.Struct(new(TeacherAssembler), "*"),
	wire.Bind(new(ITeacherAssembler), new(*TeacherAssembler)),
)

// ToTeacherVO 单个TeacherDB转TeacherVO (DB to VO)
func (a *TeacherAssembler) ToTeacherVO(db *model.Teacher) *dto.TeacherVO {
	return &dto.TeacherVO{
		ID:         db.ID,
		Name:       db.Name,
		Title:      db.Title,
		Department: mapping.Data.GetDepartmentNameByID(db.Department),
	}
}

// ToTeacherDB 单个TeacherVO转TeacherDB (VO to DB)
func (a *TeacherAssembler) ToTeacherDB(vo *dto.TeacherVO) *model.Teacher {
	return &model.Teacher{
		ID:         vo.ID,
		Name:       vo.Name,
		Title:      vo.Title,
		Department: mapping.Data.GetDepartmentIDByName(vo.Department),
	}
}

// ToTeacherVOArray TeacherDB数组转TeacherVO数组 (DB Array to VO Array)
func (a *TeacherAssembler) ToTeacherVOArray(dbs []*model.Teacher) []*dto.TeacherVO {
	vos := []*dto.TeacherVO{}
	for _, db := range dbs {
		if vo := a.ToTeacherVO(db); vo != nil {
			vos = append(vos, vo)
		}
	}
	return vos
}

// ToTeacherDBArray TeacherVO数组转TeacherDB数组 (VO Array to DB Array)
func (a *TeacherAssembler) ToTeacherDBArray(vos []*dto.TeacherVO) []*model.Teacher {
	dbs := []*model.Teacher{}
	for _, vo := range vos {

		if db := a.ToTeacherDB(vo); db != nil {
			dbs = append(dbs, db)
		}
	}
	return dbs
}
