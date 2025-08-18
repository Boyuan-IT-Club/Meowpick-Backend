package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/searchhistory"
	"github.com/google/wire"
)

type ISearchHistoryService interface {
	GetSearchHistoryByUserId(ctx context.Context, userId string) ([]*cmd.SearchHistoryVO, error)
}

type SearchHistoryService struct {
	SearchHistoryMapper *searchhistory.MongoMapper
}

var SearchHistoryServiceSet = wire.NewSet(
	wire.Struct(new(SearchHistoryService), "*"),
	wire.Bind(new(ISearchHistoryService), new(*SearchHistoryService)),
)

func (s *SearchHistoryService) GetSearchHistoryByUserId(ctx context.Context, userId string) ([]*cmd.SearchHistoryVO, error) {
	histories, err := s.SearchHistoryMapper.FindByUserID(ctx, userId)
	if err != nil {
		return nil, err
	}

	vos := make([]*cmd.SearchHistoryVO, 0, len(histories))
	for _, h := range histories {
		vo := &cmd.SearchHistoryVO{
			ID:       h.ID,
			Text:     h.Query,
			CreateAt: h.CreatedAt,
		}
		vos = append(vos, vo)
	}

	return vos, nil
}
