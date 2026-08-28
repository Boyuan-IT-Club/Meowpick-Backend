// Copyright 2026 Boyuan-IT-Club
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

package main

import "testing"

func TestSnapshotHashIncludesResolvedCourseRepair(t *testing.T) {
	department := int32(10)
	first, err := snapshotHash(nil, nil, []courseRepair{{ID: "course", Department: &department}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	department = 11
	second, err := snapshotHash(nil, nil, []courseRepair{{ID: "course", Department: &department}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("snapshot hash did not change when the proposal-derived repair changed")
	}
}

func TestSnapshotHashIsOrderIndependent(t *testing.T) {
	first, err := snapshotHash(
		[]legacyMapping{{Name: "B", Code: 2}, {Name: "A", Code: 1}},
		[]brokenCourse{{ID: "B"}, {ID: "A"}},
		nil, nil, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := snapshotHash(
		[]legacyMapping{{Name: "A", Code: 1}, {Name: "B", Code: 2}},
		[]brokenCourse{{ID: "A"}, {ID: "B"}},
		nil, nil, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("snapshot hash depends on MongoDB iteration order")
	}
}
