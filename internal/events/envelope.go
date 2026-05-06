package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Envelope is a CloudEvents-inspired JSON shape for external consumers.
type Envelope struct {
	SpecVersion string         `json:"specversion"`
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Source      string         `json:"source"`
	Time        time.Time      `json:"time"`
	Data        map[string]any `json:"data"`
}

// NewEnvelope builds an envelope with a fresh id and RFC3339Nano time.
func NewEnvelope(source string, typ Type, data map[string]any) *Envelope {
	if data == nil {
		data = map[string]any{}
	}
	return &Envelope{
		SpecVersion: SpecVersion,
		ID:          uuid.NewString(),
		Type:        string(typ),
		Source:      source,
		Time:        time.Now().UTC(),
		Data:        data,
	}
}

// MarshalJSONLine returns one NDJSON line (newline-terminated).
func MarshalJSONLine(e *Envelope) ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(b)+1)
	copy(out, b)
	out[len(b)] = '\n'
	return out, nil
}
