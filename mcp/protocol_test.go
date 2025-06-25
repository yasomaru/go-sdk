// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"encoding/json"
	"maps"
	"testing"
)

func TestParamsMeta(t *testing.T) {
	// Verify some properties of the Meta field of Params structs.
	// We use CallToolParams for the test, but the Meta setup of all params types
	// is identical so they should all behave the same.

	toJSON := func(x any) string {
		data, err := json.Marshal(x)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	meta := map[string]any{"m": 1}

	// You can set the embedded Meta field to a literal map.
	p := &CallToolParams{
		Meta: meta,
		Name: "name",
	}

	// The Meta field marshals properly when it's present.
	if g, w := toJSON(p), `{"_meta":{"m":1},"name":"name"}`; g != w {
		t.Errorf("got %s, want %s", g, w)
	}
	// ... and when it's absent.
	p2 := &CallToolParams{Name: "n"}
	if g, w := toJSON(p2), `{"name":"n"}`; g != w {
		t.Errorf("got %s, want %s", g, w)
	}

	// The GetMeta and SetMeta functions work as expected.
	if g := p.GetMeta(); !maps.Equal(g, meta) {
		t.Errorf("got %+v, want %+v", g, meta)
	}

	meta2 := map[string]any{"x": 2}
	p.SetMeta(meta2)
	if g := p.GetMeta(); !maps.Equal(g, meta2) {
		t.Errorf("got %+v, want %+v", g, meta2)
	}

	// The GetProgressToken and SetProgressToken methods work as expected.
	if g := p.GetProgressToken(); g != nil {
		t.Errorf("got %v, want nil", g)
	}

	p.SetProgressToken("t")
	if g := p.GetProgressToken(); g != "t" {
		t.Errorf("got %v, want `t`", g)
	}

	// You can set a progress token to an int, int32 or int64.
	p.SetProgressToken(int(1))
	p.SetProgressToken(int32(1))
	p.SetProgressToken(int64(1))
}
