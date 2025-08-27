package util

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SetQueryParam(page int, pageSize int) *cmd.QueryParam {
	return &cmd.QueryParam{page, pageSize}
}

func GetFindOptions(param *cmd.QueryParam) *options.FindOptions {
	findOptions := options.Find()
	findOptions.SetSkip(int64((param.Page - 1) * param.PageSize))
	findOptions.SetLimit(int64(param.PageSize))
	findOptions.SetSort(bson.D{{"createdAt", -1}})
	return findOptions
}

func CheckPage(page *int, pageSize *int) {
	if *page <= 0 {
		*page = 1
	}
	if *pageSize <= 0 || *pageSize > 100 {
		*pageSize = 10
	}
}
