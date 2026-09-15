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
	"testing"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
)

func TestIsDebugSignIn(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.Config
		verifyCode string
		want       bool
	}{
		{name: "nil config"},
		{name: "disabled", cfg: debugLoginConfig("local", false)},
		{name: "production", cfg: debugLoginConfig("pro", true), verifyCode: "secret"},
		{name: "wrong code", cfg: debugLoginConfig("local", true), verifyCode: "wrong"},
		{name: "matching local code", cfg: debugLoginConfig("local", true), verifyCode: "secret", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDebugSignIn(tt.cfg, tt.verifyCode); got != tt.want {
				t.Fatalf("isDebugSignIn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func debugLoginConfig(state string, enabled bool) *config.Config {
	return &config.Config{
		State: state,
		DebugLogin: config.DebugLogin{
			Enabled:    enabled,
			VerifyCode: "secret",
			OpenID:     "debug-openid",
		},
	}
}
