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

package service

import (
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/google/wire"
)

var _ ITeacherService = (*TeacherService)(nil)

type ITeacherService interface {
	AddNewTeacher(ctx context.Context, req *dto.CreateTeacherReq) (*dto.CreateTeacherResp, error)
}

type TeacherService struct {
	CourseRepo       *repo.CourseRepo
	CommentRepo      *repo.CommentRepo
	UserRepo         *repo.UserRepo
	TeacherRepo      *repo.TeacherRepo
	CourseAssembler  *assembler.CourseAssembler
	TeacherAssembler *assembler.TeacherAssembler
}

var TeacherServiceSet = wire.NewSet(
	wire.Struct(new(TeacherService), "*"),
	wire.Bind(new(ITeacherService), new(*TeacherService)),
)

func (s *TeacherService) AddNewTeacher(ctx context.Context, req *dto.CreateTeacherReq) (*dto.CreateTeacherResp, error) {
	// 鉴权
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		log.Error("Get user Id failed")
		return nil, errorx.ErrTokenInvalid
	}
	if admin, _ := s.UserRepo.IsAdmin(ctx, userID); !admin {
		return nil, errorx.ErrUserNotAdmin
	}

	// 如果拥有管理员权限，继续向下执行添加教师的逻辑
	teacherVO := &dto.TeacherVO{
		Name:       req.Name,
		Title:      req.Title,
		Department: req.Department,
	}
	dbTeacher, err := s.TeacherAssembler.ToTeacher(ctx, teacherVO)
	if err != nil {
		log.Error("TeacherVO To dbTeacher err:", teacherVO, err)
	}
	// 防重
	existingTeacher, err := s.TeacherRepo.FindByID(ctx, dbTeacher.ID)
	if err != nil && existingTeacher != nil {
		return nil, errorx.ErrTeacherDuplicate
	}

	// 增加教师
	teacherId, err := s.TeacherRepo.Insert(ctx, dbTeacher)
	if err != nil {
		log.Error("Add New Teacher failed", err)
		return nil, err
	}
	teacherVO.ID = teacherId

	return &dto.CreateTeacherResp{Resp: dto.Success(), TeacherVO: teacherVO}, nil
}
