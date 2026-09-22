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

package config

import "testing"

func TestValidateWeApp(t *testing.T) {
	tests := []struct {
		name    string
		weApp   WeApp
		wantErr bool
	}{
		{name: "complete", weApp: WeApp{AppID: "wx-app", AppSecret: "secret"}},
		{name: "missing app id", weApp: WeApp{AppSecret: "secret"}, wantErr: true},
		{name: "missing secret", weApp: WeApp{AppID: "wx-app"}, wantErr: true},
		{name: "blank values", weApp: WeApp{AppID: " ", AppSecret: "\t"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWeApp(tt.weApp)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateWeApp() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDebugLogin(t *testing.T) {
	tests := []struct {
		name       string
		debugLogin DebugLogin
		wantErr    bool
	}{
		{name: "disabled"},
		{name: "enabled", debugLogin: DebugLogin{Enabled: true, VerifyCode: "code", OpenID: "openid"}},
		{name: "missing verify code", debugLogin: DebugLogin{Enabled: true, OpenID: "openid"}, wantErr: true},
		{name: "missing open id", debugLogin: DebugLogin{Enabled: true, VerifyCode: "code"}, wantErr: true},
		{name: "blank values", debugLogin: DebugLogin{Enabled: true, VerifyCode: " ", OpenID: "\t"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDebugLogin(tt.debugLogin)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateDebugLogin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
