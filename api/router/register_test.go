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
		if key == "GET /api/course/departs" {
			t.Fatal("removed GET /api/course/departs route is still registered")
		}
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

func TestProposalSearchUsesSuggestRouteOnly(t *testing.T) {
	var suggestFound bool
	for _, route := range SetupRoutes().Routes() {
		if route.Method != "GET" {
			continue
		}
		switch route.Path {
		case "/api/proposal/suggest":
			suggestFound = true
		case "/api/proposal/filter":
			t.Fatal("deprecated GET /api/proposal/filter route is still registered")
		}
	}
	if !suggestFound {
		t.Fatal("GET /api/proposal/suggest route is not registered")
	}
}

func TestProposalResubmitRouteIsRegistered(t *testing.T) {
	for _, route := range SetupRoutes().Routes() {
		if route.Method == "POST" && route.Path == "/api/proposal/:proposalId/resubmit" {
			return
		}
	}
	t.Fatal("POST /api/proposal/:proposalId/resubmit route is not registered")
}
