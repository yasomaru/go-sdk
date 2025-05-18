// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// !+
package main

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type HiParams struct {
	Name string `json:"name"`
}

func SayHi(ctx context.Context, cc *mcp.ServerSession, params *HiParams) ([]*mcp.Content, error) {
	return []*mcp.Content{
		mcp.NewTextContent("Hi " + params.Name),
	}, nil
}

func main() {
	// Create a server with a single tool.
	server := mcp.NewServer("greeter", "v1.0.0", nil)
	server.AddTools(mcp.NewTool("greet", "say hi", SayHi))
	// Run the server over stdin/stdout, until the client diconnects
	_ = server.Run(context.Background(), mcp.NewStdIOTransport())
}

// !-
