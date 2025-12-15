package service

import "github.com/google/wire"

var _ IProposalService = (*ProposalService)(nil)

type IProposalService interface {
}

type ProposalService struct {
}

var ProposalServiceSet = wire.NewSet(
	wire.Struct(new(ProposalService), "*"),
	wire.Bind(new(IProposalService), new(*ProposalService)),
)
