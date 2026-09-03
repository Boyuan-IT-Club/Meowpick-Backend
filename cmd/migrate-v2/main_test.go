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

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConnectionFromConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("Mongo:\n  URL: mongodb://test-mongo:27017/meowpick\n  DB: meowpick\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	uri, database, err := resolveConnection("", "", path)
	if err != nil {
		t.Fatal(err)
	}
	if uri != "mongodb://test-mongo:27017/meowpick" || database != "meowpick" {
		t.Fatalf("unexpected connection: uri=%q database=%q", uri, database)
	}
}

func TestResolveConnectionExplicitValuesOverrideConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("Mongo:\n  URL: mongodb://from-config:27017/config-db\n  DB: config-db\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	uri, database, err := resolveConnection("mongodb://explicit:27017/explicit-db", "explicit-db", path)
	if err != nil {
		t.Fatal(err)
	}
	if uri != "mongodb://explicit:27017/explicit-db" || database != "explicit-db" {
		t.Fatalf("explicit values were not preserved: uri=%q database=%q", uri, database)
	}
}

func TestValidateRequiredTarget(t *testing.T) {
	if err := validateRequiredTarget("mongodb://test-mongo:27017/meowpick?replicaSet=rs0", "meowpick", "test-mongo", 27017, "meowpick"); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string][2]string{
		"host":     {"mongodb://prod-mongo:27017/meowpick", "meowpick"},
		"port":     {"mongodb://test-mongo:27018/meowpick", "meowpick"},
		"database": {"mongodb://test-mongo:27017/prod", "prod"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateRequiredTarget(target[0], target[1], "test-mongo", 27017, "meowpick"); err == nil {
				t.Fatal("expected target validation to fail")
			}
		})
	}
}

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
