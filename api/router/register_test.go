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

package router

import "testing"

func TestGetUsernameByUserIDRoute(t *testing.T) {
	for _, route := range SetupRoutes().Routes() {
		if route.Method == "GET" && route.Path == "/api/user/:userId/username" {
			return
		}
	}
	t.Fatal("GET /api/user/:userId/username route is not registered")
}

func TestCorrectedRoutesAreRegistered(t *testing.T) {
	want := map[string]bool{
		"POST /api/teacher/add":       false,
		"POST /api/changelog/list":    false,
		"GET /api/course/departments": false,
	}
	for _, route := range SetupRoutes().Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("route %s is not registered", route)
		}
	}
}
