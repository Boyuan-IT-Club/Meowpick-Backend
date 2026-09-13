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

import (
	"context"
	"errors"
	"testing"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/wechatsecurity"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
)

type fakeContentSecurityClient struct {
	result *wechatsecurity.TextCheckResult
	err    error
}

func (f *fakeContentSecurityClient) CheckText(context.Context, wechatsecurity.TextCheckRequest) (*wechatsecurity.TextCheckResult, error) {
	return f.result, f.err
}

func TestContentModerationServiceCheckText(t *testing.T) {
	tests := []struct {
		name      string
		result    *wechatsecurity.TextCheckResult
		clientErr error
		wantCode  int32
	}{
		{name: "pass", result: &wechatsecurity.TextCheckResult{Suggest: "pass", Label: 100, TraceID: "trace-pass"}},
		{name: "review", result: &wechatsecurity.TextCheckResult{Suggest: "review", Label: 20001, TraceID: "trace-review"}, wantCode: errno.ErrContentModerationRejected},
		{name: "risky", result: &wechatsecurity.TextCheckResult{Suggest: "risky", Label: 20002, TraceID: "trace-risky"}, wantCode: errno.ErrContentModerationRejected},
		{name: "unknown suggestion", result: &wechatsecurity.TextCheckResult{Suggest: "unknown", TraceID: "trace-unknown"}, wantCode: errno.ErrContentModerationUnavailable},
		{name: "empty result", wantCode: errno.ErrContentModerationUnavailable},
		{name: "client failure", clientErr: errors.New("provider unavailable"), wantCode: errno.ErrContentModerationUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ContentModerationService{Client: &fakeContentSecurityClient{result: tt.result, err: tt.clientErr}}
			err := service.CheckText(context.Background(), wechatsecurity.TextCheckRequest{
				OpenID: "openid", Scene: wechatsecurity.SceneComment, Content: "comment",
			})
			if tt.wantCode == 0 {
				if err != nil {
					t.Fatalf("CheckText() error = %v", err)
				}
				return
			}
			var statusErr errorx.StatusError
			if !errors.As(err, &statusErr) {
				t.Fatalf("CheckText() error = %T, want StatusError", err)
			}
			if statusErr.Code() != tt.wantCode {
				t.Fatalf("code = %d, want %d", statusErr.Code(), tt.wantCode)
			}
		})
	}
}
