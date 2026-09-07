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

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"go.mongodb.org/mongo-driver/bson"
)

func TestTeacherReferenceFiltersExcludeDeletedRecords(t *testing.T) {
	wantDeleted := bson.M{"$ne": true}
	courseFilter := activeTeacherReferenceFilter("teacher-1")
	if courseFilter[consts.TeacherIDs] != "teacher-1" || !reflect.DeepEqual(courseFilter[consts.Deleted], wantDeleted) {
		t.Fatalf("activeTeacherReferenceFilter() = %#v", courseFilter)
	}
	proposalFilter := proposalTeacherReferenceFilter("teacher-1")
	if proposalFilter[consts.PathCourseTeacherID] != "teacher-1" || !reflect.DeepEqual(proposalFilter[consts.Deleted], wantDeleted) {
		t.Fatalf("proposalTeacherReferenceFilter() = %#v", proposalFilter)
	}
}

func TestMappingReferenceFiltersExcludeDeletedRecords(t *testing.T) {
	wantDeleted := bson.M{"$ne": true}

	courseDepartment, err := activeMappingReferenceFilter(model.MappingTypeDepartment, 12)
	if err != nil {
		t.Fatal(err)
	}
	if courseDepartment[consts.Department] != int32(12) || !reflect.DeepEqual(courseDepartment[consts.Deleted], wantDeleted) {
		t.Fatalf("activeMappingReferenceFilter(department) = %#v", courseDepartment)
	}
}

func TestActiveInt32References(t *testing.T) {
	values := []interface{}{int32(2), int64(3), "4", int64(1 << 40)}
	want := []int32{2, 3}
	if got := activeInt32References(values); !reflect.DeepEqual(got, want) {
		t.Fatalf("activeInt32References() = %v, want %v", got, want)
	}
}
