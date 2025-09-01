package util

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func FindPageOption(param cmd.IPageParam) *options.FindOptions {
	page, size := param.UnWrap()
	findOptions := options.Find()
	findOptions.SetSkip((page - 1) * size)
	findOptions.SetLimit(size)

	return findOptions
}

func DSort(s string, i int) bson.D {
	return bson.D{{s, i}}
}
