package util

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type QueryParam struct {
	page     int
	pageSize int
}

func SetQueryParam(page int, pageSize int) QueryParam {
	return QueryParam{page, pageSize}
}

func GetFindOptions(param QueryParam) *options.FindOptions {
	findOptions := options.Find()
	findOptions.SetSkip(int64((param.page - 1) * param.pageSize))
	findOptions.SetLimit(int64(param.pageSize))
	findOptions.SetSort(bson.D{{"createdAt", -1}})
	return findOptions
}

func CheckPage(query *cmd.GetCoursesReq) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}
}
