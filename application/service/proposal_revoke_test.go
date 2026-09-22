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

package service

import (
	"errors"
	"testing"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
)

func TestValidateApprovalRevocation(t *testing.T) {
	const approvedStatusID = int32(2)
	china := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, time.September, 22, 18, 20, 0, 0, china)
	tests := []struct {
		name             string
		proposal         model.Proposal
		loggedApprovalAt time.Time
		now              time.Time
		wantCode         int32
	}{
		{
			name:     "immediately after approval is allowed",
			proposal: model.Proposal{Status: approvedStatusID, UpdatedAt: now},
			now:      now,
		},
		{
			name:     "one nanosecond before twenty four hours is allowed",
			proposal: model.Proposal{Status: approvedStatusID, UpdatedAt: now.Add(-24*time.Hour + time.Nanosecond)},
			now:      now,
		},
		{
			name:     "exactly twenty four hours is rejected",
			proposal: model.Proposal{Status: approvedStatusID, UpdatedAt: now.Add(-24 * time.Hour)},
			now:      now,
			wantCode: errno.ErrProposalRevokeTimeLimitExceeded,
		},
		{
			name:     "one nanosecond after twenty four hours is rejected",
			proposal: model.Proposal{Status: approvedStatusID, UpdatedAt: now.Add(-24*time.Hour - time.Nanosecond)},
			now:      now,
			wantCode: errno.ErrProposalRevokeTimeLimitExceeded,
		},
		{
			name: "crossing midnight does not end the window",
			proposal: model.Proposal{
				Status:    approvedStatusID,
				UpdatedAt: time.Date(2026, time.September, 21, 23, 50, 0, 0, china),
			},
			now: time.Date(2026, time.September, 22, 0, 10, 0, 0, china),
		},
		{
			name:     "different time zones within the window are allowed",
			proposal: model.Proposal{Status: approvedStatusID, UpdatedAt: now.Add(-23 * time.Hour).UTC()},
			now:      now,
		},
		{
			name:     "different time zones at the deadline are rejected",
			proposal: model.Proposal{Status: approvedStatusID, UpdatedAt: now.Add(-24 * time.Hour).UTC()},
			now:      now,
			wantCode: errno.ErrProposalRevokeTimeLimitExceeded,
		},
		{
			name: "latest approval starts a fresh window for an old proposal",
			proposal: model.Proposal{
				Status:    approvedStatusID,
				CreatedAt: now.Add(-30 * 24 * time.Hour),
				UpdatedAt: now.Add(-time.Hour),
			},
			now: now,
		},
		{
			name:     "missing approval timestamp is rejected",
			proposal: model.Proposal{Status: approvedStatusID},
			now:      now,
			wantCode: errno.ErrProposalRevokeTimeLimitExceeded,
		},
		{
			name:             "legacy like cannot renew an expired approval",
			proposal:         model.Proposal{Status: approvedStatusID, UpdatedAt: now.Add(-time.Minute)},
			loggedApprovalAt: now.Add(-48 * time.Hour),
			now:              now,
			wantCode:         errno.ErrProposalRevokeTimeLimitExceeded,
		},
		{
			name:             "legacy like is rejected at the logged approval deadline",
			proposal:         model.Proposal{Status: approvedStatusID, UpdatedAt: now.Add(-time.Minute)},
			loggedApprovalAt: now.Add(-24 * time.Hour),
			now:              now,
			wantCode:         errno.ErrProposalRevokeTimeLimitExceeded,
		},
		{
			name:             "recent reapproval log starts a fresh window",
			proposal:         model.Proposal{Status: approvedStatusID, UpdatedAt: now.Add(-time.Minute)},
			loggedApprovalAt: now.Add(-time.Hour),
			now:              now,
		},
		{
			name:             "log written later does not extend the stored deadline",
			proposal:         model.Proposal{Status: approvedStatusID, UpdatedAt: now.Add(-24 * time.Hour)},
			loggedApprovalAt: now.Add(-24*time.Hour + time.Second),
			now:              now,
			wantCode:         errno.ErrProposalRevokeTimeLimitExceeded,
		},
		{
			name:             "approval log recovers a missing proposal timestamp",
			proposal:         model.Proposal{Status: approvedStatusID},
			loggedApprovalAt: now.Add(-time.Hour),
			now:              now,
		},
		{
			name:     "pending proposal is rejected even with a recent timestamp",
			proposal: model.Proposal{Status: 1, UpdatedAt: now.Add(-time.Hour)},
			now:      now,
			wantCode: errno.ErrProposalStatusNotApproved,
		},
		{
			name:     "status mismatch takes precedence over an expired timestamp",
			proposal: model.Proposal{Status: 3, UpdatedAt: now.Add(-48 * time.Hour)},
			now:      now,
			wantCode: errno.ErrProposalStatusNotApproved,
		},
		{
			name:     "status mismatch takes precedence over a missing timestamp",
			proposal: model.Proposal{Status: 3},
			now:      now,
			wantCode: errno.ErrProposalStatusNotApproved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateApprovalRevocation(&tt.proposal, approvedStatusID, tt.loggedApprovalAt, tt.now)
			if tt.wantCode == 0 {
				if err != nil {
					t.Fatalf("validateApprovalRevocation() error = %v, want nil", err)
				}
				return
			}
			var statusErr errorx.StatusError
			if !errors.As(err, &statusErr) {
				t.Fatalf("validateApprovalRevocation() error = %T (%v), want StatusError", err, err)
			}
			if got := statusErr.Code(); got != tt.wantCode {
				t.Fatalf("validateApprovalRevocation() code = %d, want %d", got, tt.wantCode)
			}
		})
	}
}
