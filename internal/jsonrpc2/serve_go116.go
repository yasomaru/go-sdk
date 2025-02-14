// Copyright 2022 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build go1.16

package jsonrpc2

import (
	"net"
)

var errClosed = net.ErrClosed
