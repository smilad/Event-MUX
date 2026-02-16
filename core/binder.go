package core

import (
	"encoding/json"
	"fmt"
)

// Binder deserializes raw message bytes into a Go value.
// Implement this interface for custom serialization formats (Protobuf, Avro, etc.).
type Binder interface {
	Bind(data []byte, v any) error
}

// JSONBinder deserializes JSON message bodies.
type JSONBinder struct{}

func (JSONBinder) Bind(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("json: %w", err)
	}
	return nil
}
