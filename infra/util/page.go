package util

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func FindPageOption(param dto.IPageParam) *options.FindOptions {
	page, size := param.UnWrap()
	ops := options.Find()
	ops.SetSkip(page * size)
	ops.SetLimit(size)

	return ops
}

func DSort(s string, i int) bson.D {
	return bson.D{{s, i}}
}
