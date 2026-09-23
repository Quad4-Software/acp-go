// SPDX-License-Identifier: 0BSD

package acp

import (
	"encoding/json"
	"fmt"
)

// ContentBlockType discriminates ContentBlock variants.
type ContentBlockType string

// Content block types.
const (
	ContentBlockText         ContentBlockType = "text"
	ContentBlockImage        ContentBlockType = "image"
	ContentBlockAudio        ContentBlockType = "audio"
	ContentBlockResourceLink ContentBlockType = "resource_link"
	ContentBlockResource     ContentBlockType = "resource"
)

// ContentBlock is a displayable unit of content exchanged in prompts,
// message chunks, and tool call output. Exactly one variant field is
// populated, matching Type. Blocks received with an unrecognized type
// are preserved in Raw.
//
// The structure is compatible with MCP content types.
type ContentBlock struct {
	Type         ContentBlockType
	Text         *TextContent
	Image        *ImageContent
	Audio        *AudioContent
	ResourceLink *ResourceLink
	Resource     *EmbeddedResource

	// Raw preserves the original JSON of an unrecognized block type.
	// Set it directly to send a non-standard block.
	Raw json.RawMessage
}

// TextContent is text that may be plain or Markdown formatted.
type TextContent struct {
	Text        string       `json:"text"`
	Annotations *Annotations `json:"annotations,omitempty"`
	Meta        Meta         `json:"_meta,omitempty"`
}

// ImageContent is base64 encoded image data.
type ImageContent struct {
	Data        string       `json:"data"`
	MimeType    string       `json:"mimeType"`
	URI         string       `json:"uri,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	Meta        Meta         `json:"_meta,omitempty"`
}

// AudioContent is base64 encoded audio data.
type AudioContent struct {
	Data        string       `json:"data"`
	MimeType    string       `json:"mimeType"`
	Annotations *Annotations `json:"annotations,omitempty"`
	Meta        Meta         `json:"_meta,omitempty"`
}

// ResourceLink references a resource the agent can access.
type ResourceLink struct {
	URI         string       `json:"uri"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	MimeType    string       `json:"mimeType,omitempty"`
	Title       string       `json:"title,omitempty"`
	Size        *int64       `json:"size,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	Meta        Meta         `json:"_meta,omitempty"`
}

// EmbeddedResource carries complete resource contents inline, avoiding
// extra round-trips.
type EmbeddedResource struct {
	Resource    ResourceContents `json:"resource"`
	Annotations *Annotations     `json:"annotations,omitempty"`
	Meta        Meta             `json:"_meta,omitempty"`
}

// ResourceContents is the payload of an EmbeddedResource: either text
// or base64 encoded binary data.
type ResourceContents struct {
	Text *TextResourceContents
	Blob *BlobResourceContents
}

// TextResourceContents holds text resource data.
type TextResourceContents struct {
	URI      string `json:"uri"`
	Text     string `json:"text"`
	MimeType string `json:"mimeType,omitempty"`
	Meta     Meta   `json:"_meta,omitempty"`
}

// BlobResourceContents holds base64 encoded binary resource data.
type BlobResourceContents struct {
	URI      string `json:"uri"`
	Blob     string `json:"blob"`
	MimeType string `json:"mimeType,omitempty"`
	Meta     Meta   `json:"_meta,omitempty"`
}

// MarshalJSON emits the populated variant.
func (r ResourceContents) MarshalJSON() ([]byte, error) {
	switch {
	case r.Text != nil:
		return json.Marshal(r.Text)
	case r.Blob != nil:
		return json.Marshal(r.Blob)
	default:
		return []byte("null"), nil
	}
}

// UnmarshalJSON decodes text or blob contents by shape.
func (r *ResourceContents) UnmarshalJSON(b []byte) error {
	var probe struct {
		Text *string `json:"text"`
		Blob *string `json:"blob"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return err
	}
	if probe.Text != nil {
		var v TextResourceContents
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		r.Text, r.Blob = &v, nil
		return nil
	}
	var v BlobResourceContents
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	r.Blob, r.Text = &v, nil
	return nil
}

// NewTextBlock returns a text content block.
func NewTextBlock(text string) ContentBlock {
	return ContentBlock{Type: ContentBlockText, Text: &TextContent{Text: text}}
}

// NewImageBlock returns an image content block.
func NewImageBlock(data, mimeType string) ContentBlock {
	return ContentBlock{Type: ContentBlockImage, Image: &ImageContent{Data: data, MimeType: mimeType}}
}

// NewAudioBlock returns an audio content block.
func NewAudioBlock(data, mimeType string) ContentBlock {
	return ContentBlock{Type: ContentBlockAudio, Audio: &AudioContent{Data: data, MimeType: mimeType}}
}

// NewResourceLinkBlock returns a resource link content block.
func NewResourceLinkBlock(uri, name string) ContentBlock {
	return ContentBlock{Type: ContentBlockResourceLink, ResourceLink: &ResourceLink{URI: uri, Name: name}}
}

// NewResourceBlock returns an embedded resource content block.
func NewResourceBlock(r ResourceContents) ContentBlock {
	return ContentBlock{Type: ContentBlockResource, Resource: &EmbeddedResource{Resource: r}}
}

// TypeOf returns the block's effective type, inferring it from the
// populated variant when Type is unset.
func (b ContentBlock) TypeOf() ContentBlockType {
	if b.Type != "" {
		return b.Type
	}
	switch {
	case b.Text != nil:
		return ContentBlockText
	case b.Image != nil:
		return ContentBlockImage
	case b.Audio != nil:
		return ContentBlockAudio
	case b.ResourceLink != nil:
		return ContentBlockResourceLink
	case b.Resource != nil:
		return ContentBlockResource
	default:
		return ""
	}
}

// MarshalJSON emits the populated variant with its type tag.
func (b ContentBlock) MarshalJSON() ([]byte, error) {
	if b.Raw != nil && b.Text == nil && b.Image == nil && b.Audio == nil &&
		b.ResourceLink == nil && b.Resource == nil {
		return b.Raw, nil
	}
	var tag string
	var v any
	switch t := b.TypeOf(); t {
	case ContentBlockText:
		tag, v = string(t), b.Text
	case ContentBlockImage:
		tag, v = string(t), b.Image
	case ContentBlockAudio:
		tag, v = string(t), b.Audio
	case ContentBlockResourceLink:
		tag, v = string(t), b.ResourceLink
	case ContentBlockResource:
		tag, v = string(t), b.Resource
	default:
		if b.Raw != nil {
			return b.Raw, nil
		}
		return nil, fmt.Errorf("acp: unknown content block type %q", b.Type)
	}
	return marshalTagged("type", tag, v)
}

// UnmarshalJSON decodes a content block by its type field.
func (b *ContentBlock) UnmarshalJSON(data []byte) error {
	tag, err := readTag(data, "type")
	if err != nil {
		return err
	}
	*b = ContentBlock{Type: ContentBlockType(tag)}
	switch b.Type {
	case ContentBlockText:
		b.Text = new(TextContent)
		err = json.Unmarshal(data, b.Text)
	case ContentBlockImage:
		b.Image = new(ImageContent)
		err = json.Unmarshal(data, b.Image)
	case ContentBlockAudio:
		b.Audio = new(AudioContent)
		err = json.Unmarshal(data, b.Audio)
	case ContentBlockResourceLink:
		b.ResourceLink = new(ResourceLink)
		err = json.Unmarshal(data, b.ResourceLink)
	case ContentBlockResource:
		b.Resource = new(EmbeddedResource)
		err = json.Unmarshal(data, b.Resource)
	default:
		b.Raw = append(json.RawMessage(nil), data...)
	}
	return err
}

// ContentChunk is a streamed piece of a message.
type ContentChunk struct {
	Content   ContentBlock `json:"content"`
	MessageID MessageID    `json:"messageId,omitempty"`
	Meta      Meta         `json:"_meta,omitempty"`
}
