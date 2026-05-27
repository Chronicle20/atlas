package seeder

import (
	"encoding/json"
	"fmt"
	"regexp"
)

type Envelope struct {
	Data EnvelopeData `json:"data"`
}

type EnvelopeData struct {
	Type          string          `json:"type"`
	ID            string          `json:"id"`
	Attributes    json.RawMessage `json:"attributes"`
	Relationships json.RawMessage `json:"relationships,omitempty"`
}

func ParseEnvelope(b []byte) (Envelope, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return Envelope{}, fmt.Errorf("parse envelope: %w", err)
	}
	dataRaw, ok := raw["data"]
	if !ok {
		return Envelope{}, fmt.Errorf("parse envelope: missing data object")
	}
	var env Envelope
	if err := json.Unmarshal(dataRaw, &env.Data); err != nil {
		return Envelope{}, fmt.Errorf("parse envelope data: %w", err)
	}
	if env.Data.Type == "" {
		return Envelope{}, fmt.Errorf("parse envelope: data.type empty")
	}
	if env.Data.ID == "" {
		return Envelope{}, fmt.Errorf("parse envelope: data.id empty")
	}
	return env, nil
}

// DecodeAttributes parses payload as a JSON:API envelope and unmarshals
// data.attributes into target. Use this from Subdomain.Decode when the
// catalog file's payload sits inline under data.attributes (the
// majority of subdomains).
//
// Subdomains whose files instead encode their payload through
// relationships + included[] (e.g. reactor-drop, where the data block
// has only relationships and the included array carries the per-drop
// attributes) should not call this helper; they should walk payload
// directly to materialize their JSONModel from included[].
func DecodeAttributes(payload []byte, target any) error {
	env, err := ParseEnvelope(payload)
	if err != nil {
		return err
	}
	if len(env.Data.Attributes) == 0 {
		return fmt.Errorf("envelope has no data.attributes")
	}
	if err := json.Unmarshal(env.Data.Attributes, target); err != nil {
		return fmt.Errorf("unmarshal attributes: %w", err)
	}
	return nil
}

func ExtractEntityID(filename string, pattern *regexp.Regexp) (string, error) {
	if pattern == nil {
		return "", fmt.Errorf("extract id: nil pattern")
	}
	m := pattern.FindStringSubmatch(filename)
	if len(m) < 2 {
		return "", fmt.Errorf("extract id: filename %q does not match pattern", filename)
	}
	return m[1], nil
}
