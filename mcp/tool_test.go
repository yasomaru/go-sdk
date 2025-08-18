// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
)

// testToolHandler is used for type inference in TestNewServerTool.
func testToolHandler[In, Out any](context.Context, *ServerRequest[*CallToolParamsFor[In]]) (*CallToolResultFor[Out], error) {
	panic("not implemented")
}

func srvTool[In, Out any](t *testing.T, tool *Tool, handler ToolHandlerFor[In, Out]) *serverTool {
	t.Helper()
	st, err := newServerTool(tool, handler)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestNewServerTool(t *testing.T) {
	type (
		Name struct {
			Name string `json:"name"`
		}
		Size struct {
			Size int `json:"size"`
		}
	)

	nameSchema := &jsonschema.Schema{
		Type:     "object",
		Required: []string{"name"},
		Properties: map[string]*jsonschema.Schema{
			"name": {Type: "string"},
		},
		AdditionalProperties: &jsonschema.Schema{Not: new(jsonschema.Schema)},
	}
	sizeSchema := &jsonschema.Schema{
		Type:     "object",
		Required: []string{"size"},
		Properties: map[string]*jsonschema.Schema{
			"size": {Type: "integer"},
		},
		AdditionalProperties: &jsonschema.Schema{Not: new(jsonschema.Schema)},
	}

	tests := []struct {
		tool            *serverTool
		wantIn, wantOut *jsonschema.Schema
	}{
		{
			srvTool(t, &Tool{Name: "basic"}, testToolHandler[Name, Size]),
			nameSchema,
			sizeSchema,
		},
		{
			srvTool(t, &Tool{
				Name:        "in untouched",
				InputSchema: &jsonschema.Schema{},
			}, testToolHandler[Name, Size]),
			&jsonschema.Schema{},
			sizeSchema,
		},
		{
			srvTool(t, &Tool{Name: "out untouched", OutputSchema: &jsonschema.Schema{}}, testToolHandler[Name, Size]),
			nameSchema,
			&jsonschema.Schema{},
		},
		{
			srvTool(t, &Tool{Name: "nil out"}, testToolHandler[Name, any]),
			nameSchema,
			nil,
		},
	}
	for _, test := range tests {
		if diff := cmp.Diff(test.wantIn, test.tool.tool.InputSchema, cmpopts.IgnoreUnexported(jsonschema.Schema{})); diff != "" {
			t.Errorf("newServerTool(%q) input schema mismatch (-want +got):\n%s", test.tool.tool.Name, diff)
		}
		if diff := cmp.Diff(test.wantOut, test.tool.tool.OutputSchema, cmpopts.IgnoreUnexported(jsonschema.Schema{})); diff != "" {
			t.Errorf("newServerTool(%q) output schema mismatch (-want +got):\n%s", test.tool.tool.Name, diff)
		}
	}
}

func TestUnmarshalSchema(t *testing.T) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"x": {Type: "integer", Default: json.RawMessage("3")},
		},
	}
	resolved, err := schema.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		t.Fatal(err)
	}

	type S struct {
		X int `json:"x"`
	}

	for _, tt := range []struct {
		data string
		v    any
		want any
	}{
		{`{"x": 1}`, new(S), &S{X: 1}},
		{`{}`, new(S), &S{X: 3}},       // default applied
		{`{"x": 0}`, new(S), &S{X: 3}}, // FAIL: should be 0. (requires double unmarshal)
		{`{"x": 1}`, new(map[string]any), &map[string]any{"x": 1.0}},
		{`{}`, new(map[string]any), &map[string]any{"x": 3.0}}, // default applied
		{`{"x": 0}`, new(map[string]any), &map[string]any{"x": 0.0}},
	} {
		raw := json.RawMessage(tt.data)
		if err := unmarshalSchema(raw, resolved, tt.v); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(tt.v, tt.want) {
			t.Errorf("got %#v, want %#v", tt.v, tt.want)
		}

	}
}

func TestToolErrorHandling(t *testing.T) {
	// Construct server and add both tools at the top level
	server := NewServer(testImpl, nil)

	// Create a tool that returns a structured error
	structuredErrorHandler := func(ctx context.Context, req *ServerRequest[*CallToolParamsFor[map[string]any]]) (*CallToolResultFor[any], error) {
		return nil, &jsonrpc2.WireError{
			Code:    CodeInvalidParams,
			Message: "internal server error",
		}
	}

	// Create a tool that returns a regular error
	regularErrorHandler := func(ctx context.Context, req *ServerRequest[*CallToolParamsFor[map[string]any]]) (*CallToolResultFor[any], error) {
		return nil, fmt.Errorf("tool execution failed")
	}

	AddTool(server, &Tool{Name: "error_tool", Description: "returns structured error"}, structuredErrorHandler)
	AddTool(server, &Tool{Name: "regular_error_tool", Description: "returns regular error"}, regularErrorHandler)

	// Connect server and client once
	ct, st := NewInMemoryTransports()
	_, err := server.Connect(context.Background(), st, nil)
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient(testImpl, nil)
	cs, err := client.Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	// Test that structured JSON-RPC errors are returned directly
	t.Run("structured_error", func(t *testing.T) {
		// Call the tool
		_, err = cs.CallTool(context.Background(), &CallToolParams{
			Name:      "error_tool",
			Arguments: map[string]any{},
		})

		// Should get the structured error directly
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var wireErr *jsonrpc2.WireError
		if !errors.As(err, &wireErr) {
			t.Fatalf("expected WireError, got %[1]T: %[1]v", err)
		}

		if wireErr.Code != CodeInvalidParams {
			t.Errorf("expected error code %d, got %d", CodeInvalidParams, wireErr.Code)
		}
	})

	// Test that regular errors are embedded in tool results
	t.Run("regular_error", func(t *testing.T) {
		// Call the tool
		result, err := cs.CallTool(context.Background(), &CallToolParams{
			Name:      "regular_error_tool",
			Arguments: map[string]any{},
		})

		// Should not get an error at the protocol level
		if err != nil {
			t.Fatalf("unexpected protocol error: %v", err)
		}

		// Should get a result with IsError=true
		if !result.IsError {
			t.Error("expected IsError=true, got false")
		}

		// Should have error message in content
		if len(result.Content) == 0 {
			t.Error("expected error content, got empty")
		}

		if textContent, ok := result.Content[0].(*TextContent); !ok {
			t.Error("expected TextContent")
		} else if !strings.Contains(textContent.Text, "tool execution failed") {
			t.Errorf("expected error message in content, got: %s", textContent.Text)
		}
	})
}
