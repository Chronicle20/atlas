// Package anniversary implements the ANNIVERSARY event (design §15.2): a
// scheduled, server-wide window during which EXP/drop rates are multiplied
// and every online character receives a buff for the duration. Unlike
// CRIMSON_BALROG, ANNIVERSARY has no voyage scope and at most one active
// occurrence tenant-wide (FR-UI4) — its ConcurrencyKey is constant.
package anniversary

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// TypeName is the definition type this handler serves, and the registry key.
const TypeName = "ANNIVERSARY"

// Config is the ANNIVERSARY definition's configuration.
type Config struct {
	ScheduledStart time.Time `json:"scheduledStart"`
	ScheduledEnd   time.Time `json:"scheduledEnd"`
	ExpMultiplier  float64   `json:"expMultiplier"`
	DropMultiplier float64   `json:"dropMultiplier"`
	BuffSourceId   int32     `json:"buffSourceId"`
}

// DecodeConfig unmarshals a raw configuration payload into Config.
func DecodeConfig(raw json.RawMessage) (Config, error) {
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("anniversary: decode configuration: %w", err)
	}
	return c, nil
}

// Validate rejects a configuration this handler cannot interpret (FR-D6).
// Each error names its field so the JSON:API error an administrator sees is
// actionable.
func (c Config) Validate() error {
	if !c.ScheduledEnd.After(c.ScheduledStart) {
		return errors.New("scheduledEnd: must be after scheduledStart")
	}
	if c.ExpMultiplier <= 0 {
		return fmt.Errorf("expMultiplier: must be greater than zero, got %v", c.ExpMultiplier)
	}
	if c.DropMultiplier <= 0 {
		return fmt.Errorf("dropMultiplier: must be greater than zero, got %v", c.DropMultiplier)
	}
	return nil
}

// OccurrenceContext carries what Start/Advance need without a follow-up
// query: the end time (Start's NextTransitionAt) and the two multipliers
// plus the buff source id.
type OccurrenceContext struct {
	ScheduledEnd   time.Time `json:"scheduledEnd"`
	ExpMultiplier  float64   `json:"expMultiplier"`
	DropMultiplier float64   `json:"dropMultiplier"`
	BuffSourceId   int32     `json:"buffSourceId"`
}

// EncodeOccurrenceContext marshals an OccurrenceContext for storage on
// registry.Seed.Context / occurrence.Model.Context.
func EncodeOccurrenceContext(oc OccurrenceContext) (json.RawMessage, error) {
	raw, err := json.Marshal(oc)
	if err != nil {
		return nil, fmt.Errorf("anniversary: encode occurrence context: %w", err)
	}
	return raw, nil
}

// DecodeOccurrenceContext unmarshals an occurrence's stored context.
func DecodeOccurrenceContext(raw json.RawMessage) (OccurrenceContext, error) {
	var oc OccurrenceContext
	if err := json.Unmarshal(raw, &oc); err != nil {
		return OccurrenceContext{}, fmt.Errorf("anniversary: decode occurrence context: %w", err)
	}
	return oc, nil
}
