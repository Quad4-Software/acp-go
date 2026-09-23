// SPDX-License-Identifier: 0BSD

package acp

import (
	"encoding/json"
	"fmt"
)

// ElicitationSchemaType is the type discriminator of an elicitation
// schema. The only defined value is "object".
type ElicitationSchemaType string

// ElicitationSchemaObject is the only defined schema type.
const ElicitationSchemaObject ElicitationSchemaType = "object"

// ElicitationSchema is a JSON Schema object describing the fields of a
// form elicitation. Properties must be primitive typed.
type ElicitationSchema struct {
	Type        ElicitationSchemaType                `json:"type,omitempty"`
	Title       string                               `json:"title,omitempty"`
	Properties  map[string]ElicitationPropertySchema `json:"properties,omitempty"`
	Required    []string                             `json:"required,omitempty"`
	Description string                               `json:"description,omitempty"`
	Meta        Meta                                 `json:"_meta,omitempty"`
}

// ElicitationPropertyType discriminates ElicitationPropertySchema
// variants.
type ElicitationPropertyType string

// Elicitation property types.
const (
	ElicitationPropString  ElicitationPropertyType = "string"
	ElicitationPropNumber  ElicitationPropertyType = "number"
	ElicitationPropInteger ElicitationPropertyType = "integer"
	ElicitationPropBoolean ElicitationPropertyType = "boolean"
	ElicitationPropArray   ElicitationPropertyType = "array"
)

// ElicitationPropertySchema describes one form field. Exactly one
// variant field is populated, matching Type.
type ElicitationPropertySchema struct {
	Type        ElicitationPropertyType
	String      *StringPropertySchema
	Number      *NumberPropertySchema
	Integer     *IntegerPropertySchema
	Boolean     *BooleanPropertySchema
	MultiSelect *MultiSelectPropertySchema

	// Raw preserves the original JSON of an unrecognized schema type.
	Raw json.RawMessage
}

// StringFormat constrains a string property value.
type StringFormat string

// String formats.
const (
	FormatEmail    StringFormat = "email"
	FormatURI      StringFormat = "uri"
	FormatDate     StringFormat = "date"
	FormatDateTime StringFormat = "date-time"
)

// EnumOption is one titled choice in an enum property.
type EnumOption struct {
	Const       string `json:"const"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Meta        Meta   `json:"_meta,omitempty"`
}

// StringPropertySchema is a string field, or a single-select when Enum
// or OneOf is set.
type StringPropertySchema struct {
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	MinLength   *int64       `json:"minLength,omitempty"`
	MaxLength   *int64       `json:"maxLength,omitempty"`
	Pattern     string       `json:"pattern,omitempty"`
	Format      StringFormat `json:"format,omitempty"`
	Default     string       `json:"default,omitempty"`
	Enum        []string     `json:"enum,omitempty"`
	OneOf       []EnumOption `json:"oneOf,omitempty"`
	Meta        Meta         `json:"_meta,omitempty"`
}

// NumberPropertySchema is a floating-point field.
type NumberPropertySchema struct {
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Minimum     *float64 `json:"minimum,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
	Default     *float64 `json:"default,omitempty"`
	Meta        Meta     `json:"_meta,omitempty"`
}

// IntegerPropertySchema is an integer field.
type IntegerPropertySchema struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Minimum     *int64 `json:"minimum,omitempty"`
	Maximum     *int64 `json:"maximum,omitempty"`
	Default     *int64 `json:"default,omitempty"`
	Meta        Meta   `json:"_meta,omitempty"`
}

// BooleanPropertySchema is a boolean field.
type BooleanPropertySchema struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Default     *bool  `json:"default,omitempty"`
	Meta        Meta   `json:"_meta,omitempty"`
}

// MultiSelectPropertySchema is a multi-select array field.
type MultiSelectPropertySchema struct {
	Title       string           `json:"title,omitempty"`
	Description string           `json:"description,omitempty"`
	MinItems    *int64           `json:"minItems,omitempty"`
	MaxItems    *int64           `json:"maxItems,omitempty"`
	Items       MultiSelectItems `json:"items"`
	Default     []string         `json:"default,omitempty"`
	Meta        Meta             `json:"_meta,omitempty"`
}

// MultiSelectItems holds either a plain enum of string values or a
// titled anyOf list.
type MultiSelectItems struct {
	Enum  []string
	AnyOf []EnumOption

	// Raw preserves the original JSON of an unrecognized items shape.
	Raw json.RawMessage
}

// MarshalJSON emits the populated variant.
func (i MultiSelectItems) MarshalJSON() ([]byte, error) {
	switch {
	case i.Enum != nil:
		return json.Marshal(struct {
			Enum []string `json:"enum"`
			Meta Meta     `json:"_meta,omitempty"`
		}{Enum: i.Enum})
	case i.AnyOf != nil:
		return json.Marshal(struct {
			AnyOf []EnumOption `json:"anyOf"`
			Meta  Meta         `json:"_meta,omitempty"`
		}{AnyOf: i.AnyOf})
	case i.Raw != nil:
		return i.Raw, nil
	default:
		return []byte("null"), nil
	}
}

// UnmarshalJSON decodes items by shape.
func (i *MultiSelectItems) UnmarshalJSON(b []byte) error {
	var probe struct {
		Enum  json.RawMessage `json:"enum"`
		AnyOf json.RawMessage `json:"anyOf"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return err
	}
	switch {
	case probe.Enum != nil:
		return json.Unmarshal(probe.Enum, &i.Enum)
	case probe.AnyOf != nil:
		return json.Unmarshal(probe.AnyOf, &i.AnyOf)
	default:
		i.Raw = append(json.RawMessage(nil), b...)
		return nil
	}
}

// MarshalJSON emits the populated variant with its type tag.
func (p ElicitationPropertySchema) MarshalJSON() ([]byte, error) {
	if p.Raw != nil && p.String == nil && p.Number == nil && p.Integer == nil &&
		p.Boolean == nil && p.MultiSelect == nil {
		return p.Raw, nil
	}
	typ := p.Type
	if typ == "" {
		switch {
		case p.String != nil:
			typ = ElicitationPropString
		case p.Number != nil:
			typ = ElicitationPropNumber
		case p.Integer != nil:
			typ = ElicitationPropInteger
		case p.Boolean != nil:
			typ = ElicitationPropBoolean
		case p.MultiSelect != nil:
			typ = ElicitationPropArray
		}
	}
	var tag string
	var v any
	switch typ {
	case ElicitationPropString:
		tag, v = string(typ), p.String
	case ElicitationPropNumber:
		tag, v = string(typ), p.Number
	case ElicitationPropInteger:
		tag, v = string(typ), p.Integer
	case ElicitationPropBoolean:
		tag, v = string(typ), p.Boolean
	case ElicitationPropArray:
		tag, v = string(typ), p.MultiSelect
	default:
		if p.Raw != nil {
			return p.Raw, nil
		}
		return nil, fmt.Errorf("acp: unknown elicitation property type %q", p.Type)
	}
	return marshalTagged("type", tag, v)
}

// UnmarshalJSON decodes a property schema by its type field.
func (p *ElicitationPropertySchema) UnmarshalJSON(data []byte) error {
	tag, err := readTag(data, "type")
	if err != nil {
		return err
	}
	*p = ElicitationPropertySchema{Type: ElicitationPropertyType(tag)}
	switch p.Type {
	case ElicitationPropString:
		p.String = new(StringPropertySchema)
		err = json.Unmarshal(data, p.String)
	case ElicitationPropNumber:
		p.Number = new(NumberPropertySchema)
		err = json.Unmarshal(data, p.Number)
	case ElicitationPropInteger:
		p.Integer = new(IntegerPropertySchema)
		err = json.Unmarshal(data, p.Integer)
	case ElicitationPropBoolean:
		p.Boolean = new(BooleanPropertySchema)
		err = json.Unmarshal(data, p.Boolean)
	case ElicitationPropArray:
		p.MultiSelect = new(MultiSelectPropertySchema)
		err = json.Unmarshal(data, p.MultiSelect)
	default:
		p.Raw = append(json.RawMessage(nil), data...)
	}
	return err
}

// ElicitationMode discriminates CreateElicitationRequest variants.
// Values starting with an underscore are custom extensions.
type ElicitationMode string

// Elicitation modes.
const (
	ElicitationModeForm ElicitationMode = "form"
	ElicitationModeURL  ElicitationMode = "url"
)

// ElicitationSessionScope ties an elicitation to a session, optionally
// to a specific tool call.
type ElicitationSessionScope struct {
	SessionID  SessionID   `json:"sessionId"`
	ToolCallID *ToolCallID `json:"toolCallId,omitempty"`
}

// ElicitationRequestScope ties an elicitation to a JSON-RPC request
// outside of a session.
type ElicitationRequestScope struct {
	RequestID RequestID `json:"requestId"`
}

// CreateElicitationRequest asks the client to collect structured input
// from the user. Mode selects the variant. The scope is Session for
// session-tied elicitations or Request for request-tied ones; exactly
// one scope field is populated.
type CreateElicitationRequest struct {
	Message string
	Meta    Meta
	Mode    ElicitationMode

	// Scope. Exactly one of Session or Request is populated.
	Session *ElicitationSessionScope
	Request *ElicitationRequestScope

	// Form mode fields.
	RequestedSchema *ElicitationSchema

	// URL mode fields.
	ElicitationID ElicitationID
	URL           string

	// Raw preserves the original JSON of an unrecognized mode.
	Raw json.RawMessage
}

// NewFormElicitation builds a session-scoped form elicitation.
func NewFormElicitation(sid SessionID, message string, schema ElicitationSchema) CreateElicitationRequest {
	return CreateElicitationRequest{
		Mode:            ElicitationModeForm,
		Message:         message,
		Session:         &ElicitationSessionScope{SessionID: sid},
		RequestedSchema: &schema,
	}
}

// NewURLElicitation builds a session-scoped URL elicitation.
func NewURLElicitation(sid SessionID, message, id, url string) CreateElicitationRequest {
	return CreateElicitationRequest{
		Mode:          ElicitationModeURL,
		Message:       message,
		Session:       &ElicitationSessionScope{SessionID: sid},
		ElicitationID: ElicitationID(id),
		URL:           url,
	}
}

// MarshalJSON emits the request with its mode tag and scope fields.
func (r CreateElicitationRequest) MarshalJSON() ([]byte, error) {
	if r.Raw != nil && r.Session == nil && r.Request == nil &&
		r.RequestedSchema == nil && r.URL == "" {
		return r.Raw, nil
	}
	m := map[string]any{"message": r.Message}
	if r.Meta != nil {
		m["_meta"] = r.Meta
	}
	if r.Session != nil {
		m["sessionId"] = r.Session.SessionID
		if r.Session.ToolCallID != nil {
			m["toolCallId"] = r.Session.ToolCallID
		}
	}
	if r.Request != nil {
		m["requestId"] = r.Request.RequestID
	}
	mode := r.Mode
	if mode == "" {
		switch {
		case r.RequestedSchema != nil:
			mode = ElicitationModeForm
		case r.URL != "":
			mode = ElicitationModeURL
		}
	}
	switch mode {
	case ElicitationModeForm:
		m["mode"] = "form"
		m["requestedSchema"] = r.RequestedSchema
	case ElicitationModeURL:
		m["mode"] = "url"
		m["elicitationId"] = r.ElicitationID
		m["url"] = r.URL
	default:
		if r.Raw != nil {
			return r.Raw, nil
		}
		return nil, fmt.Errorf("acp: unknown elicitation mode %q", r.Mode)
	}
	return json.Marshal(m)
}

// UnmarshalJSON decodes an elicitation request by its mode field.
func (r *CreateElicitationRequest) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*r = CreateElicitationRequest{}
	if v, ok := m["message"]; ok {
		if err := json.Unmarshal(v, &r.Message); err != nil {
			return err
		}
	}
	if v, ok := m["_meta"]; ok {
		if err := json.Unmarshal(v, &r.Meta); err != nil {
			return err
		}
	}
	if _, ok := m["sessionId"]; ok {
		r.Session = new(ElicitationSessionScope)
		if err := json.Unmarshal(data, r.Session); err != nil {
			return err
		}
	}
	if _, ok := m["requestId"]; ok {
		r.Request = new(ElicitationRequestScope)
		if err := json.Unmarshal(data, r.Request); err != nil {
			return err
		}
	}
	var mode string
	if v, ok := m["mode"]; ok {
		if err := json.Unmarshal(v, &mode); err != nil {
			return err
		}
	}
	r.Mode = ElicitationMode(mode)
	switch r.Mode {
	case ElicitationModeForm:
		if v, ok := m["requestedSchema"]; ok {
			r.RequestedSchema = new(ElicitationSchema)
			return json.Unmarshal(v, r.RequestedSchema)
		}
	case ElicitationModeURL:
		if v, ok := m["elicitationId"]; ok {
			if err := json.Unmarshal(v, &r.ElicitationID); err != nil {
				return err
			}
		}
		if v, ok := m["url"]; ok {
			return json.Unmarshal(v, &r.URL)
		}
	default:
		r.Raw = append(json.RawMessage(nil), data...)
	}
	return nil
}

// ElicitationAction discriminates CreateElicitationResponse variants.
// Values starting with an underscore are custom extensions.
type ElicitationAction string

// Elicitation actions.
const (
	ElicitationAccept  ElicitationAction = "accept"
	ElicitationDecline ElicitationAction = "decline"
	ElicitationCancel  ElicitationAction = "cancel"
)

// CreateElicitationResponse is the client's answer to an elicitation
// request. Action is accept, decline, cancel, or an extension value
// preserved in Raw.
type CreateElicitationResponse struct {
	Action ElicitationAction
	Meta   Meta

	// Content holds the accepted field values keyed by property name.
	// Set only for the accept action.
	Content map[string]any

	// Raw preserves the original JSON of an unrecognized action.
	Raw json.RawMessage
}

// MarshalJSON emits the response with its action tag.
func (r CreateElicitationResponse) MarshalJSON() ([]byte, error) {
	if r.Raw != nil && r.Content == nil && r.Action == "" {
		return r.Raw, nil
	}
	switch r.Action {
	case ElicitationAccept, ElicitationDecline, ElicitationCancel:
	default:
		if r.Raw != nil {
			return r.Raw, nil
		}
		return nil, fmt.Errorf("acp: unknown elicitation action %q", r.Action)
	}
	m := map[string]any{"action": string(r.Action)}
	if r.Meta != nil {
		m["_meta"] = r.Meta
	}
	if r.Action == ElicitationAccept && r.Content != nil {
		m["content"] = r.Content
	}
	return json.Marshal(m)
}

// UnmarshalJSON decodes the response by its action field.
func (r *CreateElicitationResponse) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*r = CreateElicitationResponse{}
	if v, ok := m["_meta"]; ok {
		if err := json.Unmarshal(v, &r.Meta); err != nil {
			return err
		}
	}
	if v, ok := m["content"]; ok {
		if err := json.Unmarshal(v, &r.Content); err != nil {
			return err
		}
	}
	var action string
	if v, ok := m["action"]; ok {
		if err := json.Unmarshal(v, &action); err != nil {
			return err
		}
	}
	r.Action = ElicitationAction(action)
	switch r.Action {
	case ElicitationAccept, ElicitationDecline, ElicitationCancel:
	default:
		r.Raw = append(json.RawMessage(nil), data...)
	}
	return nil
}

// CompleteElicitationNotification is sent by the client when an
// out-of-band URL elicitation completes.
type CompleteElicitationNotification struct {
	ElicitationID ElicitationID `json:"elicitationId"`
	Meta          Meta          `json:"_meta,omitempty"`
}
