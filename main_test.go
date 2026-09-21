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

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestOpenAPIServerURLSupportsRootAndPrefixedGateways(t *testing.T) {
	router := setupRouter()
	request := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /openapi.json status = %d, want %d", response.Code, http.StatusOK)
	}

	var document struct {
		Servers []struct {
			URL string `json:"url"`
		} `json:"servers"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatalf("parse OpenAPI document: %v", err)
	}
	if len(document.Servers) != 1 || document.Servers[0].URL != "./" {
		t.Fatalf("OpenAPI servers = %#v, want one relative ./ server", document.Servers)
	}

	serverURL, err := url.Parse(document.Servers[0].URL)
	if err != nil {
		t.Fatalf("parse OpenAPI server URL: %v", err)
	}
	for _, testCase := range []struct {
		name     string
		specURL  string
		expected string
	}{
		{
			name:     "dedicated domain",
			specURL:  "https://api.example.com/openapi.json",
			expected: "https://api.example.com/",
		},
		{
			name:     "Kubernetes gateway prefix",
			specURL:  "https://gateway.example.com/service-prefix/openapi.json",
			expected: "https://gateway.example.com/service-prefix/",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			documentURL, parseErr := url.Parse(testCase.specURL)
			if parseErr != nil {
				t.Fatalf("parse document URL: %v", parseErr)
			}
			if actual := documentURL.ResolveReference(serverURL).String(); actual != testCase.expected {
				t.Fatalf("resolved server URL = %q, want %q", actual, testCase.expected)
			}
		})
	}
}

func TestSwaggerUIUsesRelativeOpenAPIURL(t *testing.T) {
	router := setupRouter()
	uiRequest := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	uiResponse := httptest.NewRecorder()
	router.ServeHTTP(uiResponse, uiRequest)

	if uiResponse.Code != http.StatusOK {
		t.Fatalf("GET /swagger/index.html status = %d, want %d", uiResponse.Code, http.StatusOK)
	}

	initializerRequest := httptest.NewRequest(http.MethodGet, "/swagger/swagger-initializer.js", nil)
	initializerResponse := httptest.NewRecorder()
	router.ServeHTTP(initializerResponse, initializerRequest)
	if initializerResponse.Code != http.StatusOK {
		t.Fatalf("GET /swagger/swagger-initializer.js status = %d, want %d", initializerResponse.Code, http.StatusOK)
	}
	if !strings.Contains(initializerResponse.Body.String(), "../openapi.json") {
		t.Fatal("Swagger UI does not use the gateway-prefix-safe relative OpenAPI URL")
	}
}
