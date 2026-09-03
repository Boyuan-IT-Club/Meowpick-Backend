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

package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func TestNormalizeRequestError(t *testing.T) {
	type request struct {
		TargetType string `validate:"required,oneof=proposal comment"`
	}

	err := validator.New().Struct(request{TargetType: "bad"})
	normalized := normalizeRequestError(err)
	var statusErr errorx.StatusError
	if !errors.As(normalized, &statusErr) {
		t.Fatalf("normalizeRequestError() = %T, want errorx.StatusError", normalized)
	}
	if statusErr.Code() != errno.ErrRequestInvalid {
		t.Fatalf("code = %d, want %d", statusErr.Code(), errno.ErrRequestInvalid)
	}
	if strings.Contains(statusErr.Msg(), "oneof") {
		t.Fatalf("message leaks validator details: %q", statusErr.Msg())
	}
}

func TestPostProcessHidesInternalErrorDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	PostProcess(ctx, nil, nil, errors.New("mongodb://user:secret@example.invalid/db"))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	body := recorder.Body.String()
	if strings.Contains(body, "secret") || strings.Contains(body, "mongodb://") {
		t.Fatalf("response leaks internal error: %s", body)
	}
	if !strings.Contains(body, "internal server error") {
		t.Fatalf("response = %s, want generic error", body)
	}
}
