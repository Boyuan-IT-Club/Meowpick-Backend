package util

import (
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

func CheckPage(page *int, pageSize *int) {
	if *page <= 0 {
		*page = 1
	}
	if *pageSize <= 0 || *pageSize > 100 {
		*pageSize = 10
	}
}
