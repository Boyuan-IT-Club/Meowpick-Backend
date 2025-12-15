package repo

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/zeromicro/go-zero/core/stores/monc"
)

var _ IProposalRepo = (*ProposalRepo)(nil)

const (
	ProposalCollectionName = "proposal"
)

type IProposalRepo interface {
}

type ProposalRepo struct {
	conn *monc.Model
}

func NewProposalRepo(cfg *config.Config) *ProposalRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, ProposalCollectionName, cfg.Cache)
	return &ProposalRepo{conn: conn}
}
