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

package errno

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Frontend branches on numeric codes; unrelated errors must never share one.
func TestBusinessErrorCodesAreUnique(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int64]string{}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range parsed.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.CONST {
				continue
			}
			for _, spec := range group.Specs {
				value := spec.(*ast.ValueSpec)
				for i, name := range value.Names {
					if !strings.HasPrefix(name.Name, "Err") {
						continue
					}
					if i >= len(value.Values) {
						t.Fatalf("error code %s must be explicit", name.Name)
					}
					literal, ok := value.Values[i].(*ast.BasicLit)
					if !ok || literal.Kind != token.INT {
						t.Fatalf("error code %s must be an integer literal", name.Name)
					}
					code, err := strconv.ParseInt(literal.Value, 0, 64)
					if err != nil {
						t.Fatal(err)
					}
					if previous, exists := seen[code]; exists {
						t.Errorf("code %d shared by %s and %s", code, previous, name.Name)
					}
					seen[code] = name.Name
				}
			}
		}
	}
}
