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
	"strings"
	"testing"
	"time"
)

func TestNormalizeNickname(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "explicit empty clears", raw: "", want: ""},
		{name: "trims outer unicode whitespace", raw: " \t小猫\u3000", want: "小猫"},
		{name: "allows internal spaces and emoji", raw: "小猫 🐱", want: "小猫 🐱"},
		{name: "allows fifteen code points", raw: strings.Repeat("猫", 15), want: strings.Repeat("猫", 15)},
		{name: "rejects whitespace only", raw: " \t\u3000", wantErr: true},
		{name: "rejects sixteen code points", raw: strings.Repeat("猫", 16), wantErr: true},
		{name: "rejects newline", raw: "小\n猫", wantErr: true},
		{name: "rejects control character", raw: "小\x00猫", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeNickname(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("normalizeNickname() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeNickname() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeNickname() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrepareNicknameUpdate(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-24 * time.Hour)
	expired := now.Add(-usernameCooldown)

	t.Run("omitted is no-op", func(t *testing.T) {
		value, changedAt, err := prepareNicknameUpdate("Alice", recent, nil, now)
		if err != nil || value != nil || changedAt != nil {
			t.Fatalf("got value=%v changedAt=%v err=%v", value, changedAt, err)
		}
	})

	t.Run("exact and whitespace-only differences are no-op", func(t *testing.T) {
		requested := " Alice "
		value, changedAt, err := prepareNicknameUpdate("Alice", recent, &requested, now)
		if err != nil || value != nil || changedAt != nil {
			t.Fatalf("got value=%v changedAt=%v err=%v", value, changedAt, err)
		}
	})

	t.Run("case-only display change observes cooldown", func(t *testing.T) {
		requested := "alice"
		_, _, err := prepareNicknameUpdate("Alice", recent, &requested, now)
		if err == nil {
			t.Fatal("prepareNicknameUpdate() error = nil, want cooldown error")
		}
	})

	t.Run("case-only display change succeeds after cooldown", func(t *testing.T) {
		requested := "alice"
		value, changedAt, err := prepareNicknameUpdate("Alice", expired, &requested, now)
		if err != nil {
			t.Fatalf("prepareNicknameUpdate() error = %v", err)
		}
		if value == nil || *value != "alice" {
			t.Fatalf("value = %v, want alice", value)
		}
		if changedAt == nil || !changedAt.Equal(now) {
			t.Fatalf("changedAt = %v, want %v", changedAt, now)
		}
	})

	t.Run("clear is allowed during cooldown and preserves timestamp", func(t *testing.T) {
		requested := ""
		value, changedAt, err := prepareNicknameUpdate("Alice", recent, &requested, now)
		if err != nil {
			t.Fatalf("prepareNicknameUpdate() error = %v", err)
		}
		if value == nil || *value != "" {
			t.Fatalf("value = %v, want empty string", value)
		}
		if changedAt != nil {
			t.Fatalf("changedAt = %v, want nil", changedAt)
		}
	})
}

func TestCanEditUsernameBoundary(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	if !canEditUsername(time.Time{}, now) {
		t.Fatal("zero timestamp should be editable")
	}
	if canEditUsername(now.Add(-usernameCooldown+time.Nanosecond), now) {
		t.Fatal("timestamp younger than cooldown should not be editable")
	}
	if !canEditUsername(now.Add(-usernameCooldown), now) {
		t.Fatal("exact cooldown boundary should be editable")
	}
}

func TestGetDailyProposalLimit(t *testing.T) {
	tests := []struct {
		contribution int64
		want         int64
	}{
		{contribution: 0, want: 5},
		{contribution: 99, want: 5},
		{contribution: 100, want: 10},
		{contribution: 499, want: 10},
		{contribution: 500, want: 20},
	}
	for _, tt := range tests {
		if got := getDailyProposalLimit(tt.contribution); got != tt.want {
			t.Fatalf("getDailyProposalLimit(%d) = %d, want %d", tt.contribution, got, tt.want)
		}
	}
}
