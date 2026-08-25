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

package docs

import (
	"encoding/json"
	"testing"
)

func TestOpenAPIBearerSecurity(t *testing.T) {
	var document map[string]any
	if err := json.Unmarshal([]byte(SwaggerInfo.ReadDoc()), &document); err != nil {
		t.Fatalf("parse generated OpenAPI: %v", err)
	}

	components := requireMap(t, document, "components")
	securitySchemes := requireMap(t, components, "securitySchemes")
	bearer := requireMap(t, securitySchemes, "Bearer")
	if bearer["type"] != "http" || bearer["scheme"] != "bearer" {
		t.Fatalf("Bearer security scheme = %#v, want HTTP bearer", bearer)
	}

	security := requireSlice(t, document, "security")
	if len(security) != 1 {
		t.Fatalf("global security = %#v, want one Bearer requirement", security)
	}
	requirement, ok := security[0].(map[string]any)
	if !ok {
		t.Fatalf("global security requirement = %#v, want object", security[0])
	}
	if scopes := requireSlice(t, requirement, "Bearer"); len(scopes) != 0 {
		t.Fatalf("Bearer scopes = %#v, want empty array", scopes)
	}

	paths := requireMap(t, document, "paths")
	signIn := requireMap(t, requireMap(t, paths, "/api/auth/sign_in"), "post")
	signInSecurity := requireSlice(t, signIn, "security")
	if len(signInSecurity) != 0 {
		t.Fatalf("sign-in security = %#v, want an explicit empty array", signInSecurity)
	}

	profile := requireMap(t, requireMap(t, paths, "/api/user/profile"), "get")
	if _, ok := profile["security"]; ok {
		t.Fatalf("profile security should be omitted so it inherits global Bearer auth")
	}
}

func requireMap(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := object[key]
	if !ok {
		t.Fatalf("missing key %q in %#v", key, object)
	}
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%q = %#v, want object", key, value)
	}
	return result
}

func requireSlice(t *testing.T, object map[string]any, key string) []any {
	t.Helper()
	value, ok := object[key]
	if !ok {
		t.Fatalf("missing key %q in %#v", key, object)
	}
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("%q = %#v, want array", key, value)
	}
	return result
}
