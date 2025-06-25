// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
)

// A ContentBlock is one of a TextContent, ImageContent, AudioContent
// ResourceLink, or EmbeddedResource.
// Use [NewTextContent], [NewImageContent], [NewAudioContent], [NewResourceLink]
// or [NewResourceContents] to create one.
//
// The Type field must be one of "text", "image", "audio", "resource_link" or "resource".
// The constructors above populate this field appropriately.
// Although at most one of Text, Data, ResourceLink and Resource should be non-zero,
// consumers of ContentBlock use the Type field to determine which value to use;
// values in the other fields are ignored.
// TODO(jba,rfindley): rethink this type. Each kind (text, image, etc.) should have its own
// meta and annotations, otherwise they're duplicated for Resource and ResourceContents.
type ContentBlock struct {
	Meta         map[string]any    `json:"_meta,omitempty"`
	Type         string            `json:"type"`
	Text         string            `json:"text,omitempty"`
	MIMEType     string            `json:"mimeType,omitempty"`
	Data         []byte            `json:"data,omitempty"`
	ResourceLink *Resource         `json:"resource_link,omitempty"`
	Resource     *ResourceContents `json:"resource,omitempty"`
	Annotations  *Annotations      `json:"annotations,omitempty"`
}

func (c *ContentBlock) UnmarshalJSON(data []byte) error {
	type wireContent ContentBlock // for naive unmarshaling
	var c2 wireContent
	if err := json.Unmarshal(data, &c2); err != nil {
		return err
	}
	switch c2.Type {
	case "text", "image", "audio", "resource", "resource_link":
	default:
		return fmt.Errorf("unrecognized content type %s", c.Type)
	}
	*c = ContentBlock(c2)
	return nil
}

// NewTextContent creates a [ContentBlock] with text.
func NewTextContent(text string) *ContentBlock {
	return &ContentBlock{Type: "text", Text: text}
}

// NewImageContent creates a [ContentBlock] with image data.
func NewImageContent(data []byte, mimeType string) *ContentBlock {
	return &ContentBlock{Type: "image", Data: data, MIMEType: mimeType}
}

// NewAudioContent creates a [ContentBlock] with audio data.
func NewAudioContent(data []byte, mimeType string) *ContentBlock {
	return &ContentBlock{Type: "audio", Data: data, MIMEType: mimeType}
}

// NewResourceLink creates a [ContentBlock] with a [Resource].
func NewResourceLink(r *Resource) *ContentBlock {
	return &ContentBlock{Type: "resource_link", ResourceLink: r}
}

// NewResourceContents creates a [ContentBlock] with an embedded resource (a [ResourceContents]).
func NewResourceContents(rc *ResourceContents) *ContentBlock {
	return &ContentBlock{Type: "resource", Resource: rc}
}

// ResourceContents represents the union of the spec's {Text,Blob}ResourceContents types.
// See https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/schema/2025-03-26/schema.ts#L524-L551
// for the inheritance structure.

// A ResourceContents is either a TextResourceContents or a BlobResourceContents.
// Use [NewTextResourceContents] or [NextBlobResourceContents] to create one.
type ResourceContents struct {
	Meta     map[string]any `json:"_meta,omitempty"`
	URI      string         `json:"uri"` // resource location; must not be empty
	MIMEType string         `json:"mimeType,omitempty"`
	Text     string         `json:"text"`
	Blob     []byte         `json:"blob,omitempty"` // if nil, then text; else blob
}

func (r ResourceContents) MarshalJSON() ([]byte, error) {
	// If we could assume Go 1.24, we could use omitzero for Blob and avoid this method.
	if r.URI == "" {
		return nil, errors.New("ResourceContents missing URI")
	}
	if r.Blob == nil {
		// Text. Marshal normally.
		type wireResourceContents ResourceContents // (lacks MarshalJSON method)
		return json.Marshal((wireResourceContents)(r))
	}
	// Blob.
	if r.Text != "" {
		return nil, errors.New("ResourceContents has non-zero Text and Blob fields")
	}
	// r.Blob may be the empty slice, so marshal with an alternative definition.
	br := struct {
		URI      string `json:"uri,omitempty"`
		MIMEType string `json:"mimeType,omitempty"`
		Blob     []byte `json:"blob"`
	}{
		URI:      r.URI,
		MIMEType: r.MIMEType,
		Blob:     r.Blob,
	}
	return json.Marshal(br)
}

// NewTextResourceContents returns a [ResourceContents] containing text.
func NewTextResourceContents(uri, mimeType, text string) *ResourceContents {
	return &ResourceContents{
		URI:      uri,
		MIMEType: mimeType,
		Text:     text,
		// Blob is nil, indicating this is a TextResourceContents.
	}
}

// NewBlobResourceContents returns a [ResourceContents] containing a byte slice.
func NewBlobResourceContents(uri, mimeType string, blob []byte) *ResourceContents {
	// The only way to distinguish text from blob is a non-nil Blob field.
	if blob == nil {
		blob = []byte{}
	}
	return &ResourceContents{
		URI:      uri,
		MIMEType: mimeType,
		Blob:     blob,
	}
}
