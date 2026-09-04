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

package assembler

import (
	"testing"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
)

func TestNewCourseContributorRespectsProposalVisibility(t *testing.T) {
	proposal := &model.Proposal{
		ID:           "proposal-1",
		UserID:       "user-1",
		ShowUsername: false,
	}
	contributor := newCourseContributor(proposal, &model.User{ID: "user-1", Username: "Alice"})
	if contributor == nil {
		t.Fatal("contributor is nil")
	}
	if contributor.ProposalID != proposal.ID || contributor.ShowUsername {
		t.Fatalf("unexpected contributor metadata: %#v", contributor)
	}
	if contributor.UserID != "" || contributor.Username != "" {
		t.Fatalf("anonymous contributor leaked identity: %#v", contributor)
	}
}

func TestNewCourseContributorIncludesVisibleUsername(t *testing.T) {
	proposal := &model.Proposal{
		ID:           "proposal-1",
		UserID:       "user-1",
		ShowUsername: true,
	}
	contributor := newCourseContributor(proposal, &model.User{ID: "user-1", Username: "Alice"})
	if contributor == nil {
		t.Fatal("contributor is nil")
	}
	if contributor.ProposalID != proposal.ID || !contributor.ShowUsername ||
		contributor.UserID != proposal.UserID || contributor.Username != "Alice" {
		t.Fatalf("unexpected visible contributor: %#v", contributor)
	}
}
