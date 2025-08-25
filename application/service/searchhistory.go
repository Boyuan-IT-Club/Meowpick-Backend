package service

import (
	"context"
	"errors"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/searchhistory"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/google/wire"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"time"
)

type ISearchHistoryService interface {
	GetSearchHistoryByUserId(ctx context.Context) (*cmd.GetSearchHistoryResp, error)
	LogSearch(ctx context.Context, query string) (*cmd.Resp, error)
}

type SearchHistoryService struct {
	SearchHistoryMapper *searchhistory.MongoMapper
}

var SearchHistoryServiceSet = wire.NewSet(
	wire.Struct(new(SearchHistoryService), "*"),
	wire.Bind(new(ISearchHistoryService), new(*SearchHistoryService)),
)

func (s *SearchHistoryService) GetSearchHistoryByUserId(ctx context.Context) (*cmd.GetSearchHistoryResp, error) {
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.ErrGetUserIDFailed
	}

	histories, err := s.SearchHistoryMapper.FindByUserID(ctx, userID)
	if err != nil {
		log.CtxError(ctx, "FindByUserID failed for userID=%s: %v", userID, err)
		return nil, err
	}

	vos := make([]cmd.SearchHistoryVO, 0, len(histories))
	for _, h := range histories {
		vo := cmd.SearchHistoryVO{
			ID:       h.ID,
			Text:     h.Query,
			CreateAt: h.CreatedAt,
		}
		vos = append(vos, vo)
	}

	resp := &cmd.GetSearchHistoryResp{
		Resp:    cmd.Success(),
		History: &vos,
	}

	return resp, nil
}

func (s *SearchHistoryService) LogSearch(ctx context.Context, query string) (*cmd.Resp, error) {
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.ErrGetUserIDFailed
	}

	// 删除同名旧记录（如果存在）
	if err := s.SearchHistoryMapper.DeleteByUserIDAndQuery(ctx, userID, query); err != nil && !errors.Is(err, monc.ErrNotFound) {
		log.CtxError(ctx, "Failed to delete existing search history for userID=%s, query=%s: %v", userID, query, err)
	}

	// 插入新记录
	now := time.Now()
	newHistory := &searchhistory.SearchHistory{
		UserID:    userID,
		Query:     query,
		CreatedAt: now,
	}
	if err := s.SearchHistoryMapper.Insert(ctx, newHistory); err != nil {
		log.CtxError(ctx, "Failed to insert new search history for userId=%s, query=%s: %v", userID, query, err)
		return nil, err
	}

	// 检查总数量，超限则删除最老记录
	count, err := s.SearchHistoryMapper.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for count > consts.SearchHistoryLimit {
		if err := s.SearchHistoryMapper.DeleteOldestByUserID(ctx, userID); err != nil && !errors.Is(err, monc.ErrNotFound) {
			log.CtxError(ctx, "Failed to delete oldest search history for userID=%s: %v", userID, err)
			break
		}
		count--
	}

	return cmd.Success(), nil
}
