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

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
)

var _ ISearchHistoryService = (*SearchHistoryService)(nil)

type ISearchHistoryService interface {
	GetSearchHistoryByUserId(ctx context.Context) (*dto.GetSearchHistoriesResp, error)
	LogSearch(ctx context.Context, query string) error
}

type SearchHistoryService struct {
	SearchHistoryRepo *repo.SearchHistoryRepo
}

var SearchHistoryServiceSet = wire.NewSet(
	wire.Struct(new(SearchHistoryService), "*"),
	wire.Bind(new(ISearchHistoryService), new(*SearchHistoryService)),
)

func (s *SearchHistoryService) GetSearchHistoryByUserId(ctx context.Context) (*dto.GetSearchHistoriesResp, error) {
	// 鉴权
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 查询获得15条历史记录
	histories, err := s.SearchHistoryRepo.FindByUserID(ctx, userID)
	if err != nil {
		logs.CtxErrorf(ctx, "SearchHistoryRepo FindByUserID error: %v", err)
		return nil, err
	}

	// 转换为VO
	vos := make([]*dto.SearchHistoryVO, 0, len(histories))
	for _, h := range histories {
		vo := &dto.SearchHistoryVO{
			ID:        h.ID,
			Query:     h.Query,
			CreatedAt: h.CreatedAt,
		}
		vos = append(vos, vo)
	}

	return &dto.GetSearchHistoriesResp{
		Resp:      dto.Success(),
		Histories: vos,
	}, nil
}

func (s *SearchHistoryService) LogSearch(ctx context.Context, query string) error {
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		return errorx.ErrGetUserIDFailed
	}

	if err := s.SearchHistoryRepo.UpsertByUserIDAndQuery(ctx, userID, query); err != nil {
		log.CtxError(ctx, "Upsert search history failed: %v", err)
		return errorx.ErrUpdateFailed
	}

	count, err := s.SearchHistoryRepo.CountByUserID(ctx, userID)
	if err != nil {
		log.CtxError(ctx, "Count search history failed: %v", err)
		return errorx.ErrCountFailed
	}

	if count > consts.SearchHistoryLimit {
		if err := s.SearchHistoryRepo.DeleteOldestByUserID(ctx, userID); err != nil {
			log.CtxError(ctx, "Failed to delete oldest search history: %v", err)
		}
	}

	return nil
}
