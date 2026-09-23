// SPDX-License-Identifier: 0BSD

package acp

import (
	"encoding/json"
	"fmt"
)

// injectTag merges the discriminator field key:value into the JSON
// object raw and returns the result. raw must decode to an object.
func injectTag(raw []byte, key, value string) ([]byte, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]json.RawMessage{}
	}
	tb, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	m[key] = tb
	return json.Marshal(m)
}

// marshalTagged marshals v and injects the discriminator field.
// A nil v marshals to an object containing only the tag.
func marshalTagged(key, value string, v any) ([]byte, error) {
	if v == nil {
		return json.Marshal(map[string]string{key: value})
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return injectTag(raw, key, value)
}

// readTag extracts the string discriminator field from a JSON object.
func readTag(raw []byte, key string) (string, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", err
	}
	tb, ok := m[key]
	if !ok {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(tb, &s); err != nil {
		return "", fmt.Errorf("acp: discriminator %q is not a string", key)
	}
	return s, nil
}
