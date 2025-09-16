package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/searchhistory"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/google/wire"
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

	vos := make([]*cmd.SearchHistoryVO, 0, len(histories))
	for _, h := range histories {
		vo := &cmd.SearchHistoryVO{
			ID:        h.ID,
			Query:     h.Query,
			CreatedAt: h.CreatedAt,
		}
		vos = append(vos, vo)
	}

	resp := &cmd.GetSearchHistoryResp{
		Resp:    cmd.Success(),
		History: vos,
	}

	return resp, nil
}

func (s *SearchHistoryService) LogSearch(ctx context.Context, query string) (*cmd.Resp, error) {
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.ErrGetUserIDFailed
	}

	if err := s.SearchHistoryMapper.UpsertByUserIDAndQuery(ctx, userID, query); err != nil {
		log.CtxError(ctx, "Upsert search history failed: %v", err)
		return nil, errorx.ErrUpdateFailed
	}

	count, err := s.SearchHistoryMapper.CountByUserID(ctx, userID)
	if err != nil {
		log.CtxError(ctx, "Count search history failed: %v", err)
		return nil, errorx.ErrCountFailed
	}

	if count > consts.SearchHistoryLimit {
		if err := s.SearchHistoryMapper.DeleteOldestByUserID(ctx, userID); err != nil {
			log.CtxError(ctx, "Failed to delete oldest search history: %v", err)
		}
	}

	return cmd.Success(), nil
}
