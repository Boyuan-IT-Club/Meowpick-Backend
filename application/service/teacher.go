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
	"errors"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ ITeacherService = (*TeacherService)(nil)

type ITeacherService interface {
	CreateTeacher(ctx context.Context, req *dto.CreateTeacherReq) (*dto.CreateTeacherResp, error)
}

type TeacherService struct {
	UserRepo         *repo.UserRepo
	TeacherRepo      *repo.TeacherRepo
	TeacherAssembler *assembler.TeacherAssembler
}

var TeacherServiceSet = wire.NewSet(
	wire.Struct(new(TeacherService), "*"),
	wire.Bind(new(ITeacherService), new(*TeacherService)),
)

// CreateTeacher 创建教师
func (s *TeacherService) CreateTeacher(ctx context.Context, req *dto.CreateTeacherReq) (*dto.CreateTeacherResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}
	if admin, err := s.UserRepo.IsAdminByID(ctx, userId); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, errorx.WrapByCode(err, errno.ErrUserNotFound,
				errorx.KV("key", consts.CtxUserID), errorx.KV("value", userId))
		}
		return nil, errorx.WrapByCode(err, errno.ErrUserFindFailed,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userId))
	} else if !admin {
		return nil, errorx.New(errno.ErrUserNotAdmin, errorx.KV("id", userId))
	}

	// 构造教师实体
	vo := &dto.TeacherVO{
		ID:         primitive.NewObjectID().Hex(),
		Name:       req.Name,
		Title:      req.Title,
		Department: req.Department,
	}

	// 转换为DB
	teacher := s.TeacherAssembler.ToTeacherDB(ctx, vo)

	// 防重 TODO：名字 职称是否重复
	if exist, err := s.TeacherRepo.IsExistByID(ctx, teacher.ID); err != nil {
		logs.CtxErrorf(ctx, "[TeacherRepo] [IsExistByID] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrTeacherExistsFailed, errorx.KV("name", teacher.Name))
	} else if exist {
		return nil, errorx.New(errno.ErrTeacherExist, errorx.KV("name", teacher.Name))
	}

	// 增加教师
	if err := s.TeacherRepo.Insert(ctx, teacher); err != nil {
		logs.CtxErrorf(ctx, "[TeacherRepo] [Insert] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrTeacherInsertFailed, errorx.KV("name", teacher.Name))
	}

	return &dto.CreateTeacherResp{Resp: dto.Success(), TeacherVO: vo}, nil
}
