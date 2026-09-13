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

package service

import "testing"

func TestProposalDetailsVisible(t *testing.T) {
	const (
		pending  = int32(1)
		approved = int32(2)
		rejected = int32(3)
	)
	tests := []struct {
		name      string
		status    int32
		isCreator bool
		isAdmin   bool
		want      bool
	}{
		{name: "approved is public", status: approved, want: true},
		{name: "creator sees pending", status: pending, isCreator: true, want: true},
		{name: "creator sees rejected", status: rejected, isCreator: true, want: true},
		{name: "admin sees pending", status: pending, isAdmin: true, want: true},
		{name: "admin sees rejected", status: rejected, isAdmin: true, want: true},
		{name: "other user cannot see pending", status: pending, want: false},
		{name: "other user cannot see rejected", status: rejected, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := proposalDetailsVisible(tt.status, approved, tt.isCreator, tt.isAdmin); got != tt.want {
				t.Fatalf("proposalDetailsVisible() = %v, want %v", got, tt.want)
			}
		})
	}
}
