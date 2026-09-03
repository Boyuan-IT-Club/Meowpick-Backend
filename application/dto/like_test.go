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

package dto

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestToggleLikeReqTargetTypeValidation(t *testing.T) {
	validate := validator.New()
	validate.SetTagName("binding")
	for _, targetType := range []string{"proposal", "comment"} {
		if err := validate.Struct(ToggleLikeReq{TargetType: targetType}); err != nil {
			t.Fatalf("targetType %q rejected: %v", targetType, err)
		}
	}
	for _, targetType := range []string{"", "bad", "course"} {
		if err := validate.Struct(ToggleLikeReq{TargetType: targetType}); err == nil {
			t.Fatalf("targetType %q accepted, want validation error", targetType)
		}
	}
}
