package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
)

// TODO: 配置文件，我们暂时在这里写死。
var departmentIDToName = map[int32]string{
	1: "计算机科学与技术学院",
	2: "马克思主义学院",
	3: "软件工程学院",
}
var categoryIDToName = map[int32]string{
	101: "专业必修课",
	102: "思政类",
}

type ISearchService interface {
	Search(ctx context.Context, req *cmd.SearchReq) (*cmd.SearchResp, error)
}
