// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

func assert(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}

func is[T any](v any) bool {
	_, ok := v.(T)
	return ok
}
