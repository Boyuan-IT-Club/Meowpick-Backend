// Copyright 2025 Boyuan-IT-Club
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repo

import (
	"reflect"
	"testing"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"go.mongodb.org/mongo-driver/bson"
)

func TestCommentOwnerDeleteFilter(t *testing.T) {
	want := bson.M{
		consts.ID:      "comment-1",
		consts.UserID:  "user-1",
		consts.Deleted: bson.M{"$ne": true},
	}
	if got := commentOwnerDeleteFilter("comment-1", "user-1"); !reflect.DeepEqual(got, want) {
		t.Fatalf("commentOwnerDeleteFilter() = %#v, want %#v", got, want)
	}
}

func TestCommentAdminDeleteFilter(t *testing.T) {
	want := bson.M{consts.ID: "comment-1", consts.Deleted: bson.M{"$ne": true}}
	if got := commentAdminDeleteFilter("comment-1"); !reflect.DeepEqual(got, want) {
		t.Fatalf("administrator delete filter = %#v, want %#v", got, want)
	}
}
