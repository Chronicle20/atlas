package effect

import (
	"encoding/json"
	"testing"
)

// TestExtract_DotFields_RoundTrip asserts the three DoT fields survive the
// atlas-data REST payload -> atlas-channel effect.Model hydration with their
// millisecond values intact. atlas-data converts dotInterval/dotTime from WZ
// seconds to ms at its reader (task-200 FR-1.2); the channel must NOT
// re-scale.
func TestExtract_DotFields_RoundTrip(t *testing.T) {
	const payload = `{"dot":105,"dotInterval":1000,"dotTime":4000}`

	var rm RestModel
	if err := json.Unmarshal([]byte(payload), &rm); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Dot() != 105 {
		t.Fatalf("m.Dot() = %d, want 105", m.Dot())
	}
	if m.DotInterval() != 1000 {
		t.Fatalf("m.DotInterval() = %d, want 1000 (ms)", m.DotInterval())
	}
	if m.DotTime() != 4000 {
		t.Fatalf("m.DotTime() = %d, want 4000 (ms)", m.DotTime())
	}
}

// TestExtract_DotFields_AbsentDefaultToZero asserts a payload without the DoT
// keys hydrates to zeros rather than failing -- which is the state of every
// provisioned tenant today (task-200 design §2.1).
func TestExtract_DotFields_AbsentDefaultToZero(t *testing.T) {
	const payload = `{"duration":4000}`

	var rm RestModel
	if err := json.Unmarshal([]byte(payload), &rm); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Dot() != 0 || m.DotInterval() != 0 || m.DotTime() != 0 {
		t.Fatalf("dot fields = (%d,%d,%d), want (0,0,0)", m.Dot(), m.DotInterval(), m.DotTime())
	}
}
