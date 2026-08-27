package configuration

import (
	"reflect"
	"testing"
)

func TestExtractFoldsZeroKnobsToDefaults(t *testing.T) {
	m := Extract(RestModel{})
	d := DefaultConfig()
	if m.MaxPerMap() != d.MaxPerMap() {
		t.Errorf("MaxPerMap = %d, want %d", m.MaxPerMap(), d.MaxPerMap())
	}
	if m.MaxMessageLength() != d.MaxMessageLength() {
		t.Errorf("MaxMessageLength = %d, want %d", m.MaxMessageLength(), d.MaxMessageLength())
	}
	if len(m.BlockedMapPrefixes()) != len(d.BlockedMapPrefixes()) {
		t.Errorf("BlockedMapPrefixes = %v, want %v", m.BlockedMapPrefixes(), d.BlockedMapPrefixes())
	}
}

func TestExtractKeepsProvidedKnobs(t *testing.T) {
	m := Extract(RestModel{MaxPerMap: 3, MaxMessageLength: 40, BlockedMapPrefixes: []uint32{91, 92}})
	if m.MaxPerMap() != 3 || m.MaxMessageLength() != 40 {
		t.Errorf("got maxPerMap=%d maxMessageLength=%d", m.MaxPerMap(), m.MaxMessageLength())
	}
	if len(m.BlockedMapPrefixes()) != 2 {
		t.Errorf("BlockedMapPrefixes = %v", m.BlockedMapPrefixes())
	}
}

// The client's own rule is GetCurFieldID() / 10000000 == 91 -> refuse
// (CWvsContext::SendConsumeCashItemUseRequest case 18, gms_v95 @0x9ed017).
// IsMapBlocked mirrors that arithmetic exactly.
func TestIsMapBlockedMirrorsClientArithmetic(t *testing.T) {
	m := DefaultConfig()
	if !m.IsMapBlocked(910000000) {
		t.Error("910000000 (Free Market entrance) should be blocked")
	}
	if !m.IsMapBlocked(919999999) {
		t.Error("919999999 (top of the FM range) should be blocked")
	}
	if m.IsMapBlocked(909999999) {
		t.Error("909999999 is below the FM range and must not be blocked")
	}
	if m.IsMapBlocked(920000000) {
		t.Error("920000000 is above the FM range and must not be blocked")
	}
	if m.IsMapBlocked(104040000) {
		t.Error("an ordinary field must not be blocked")
	}
}

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		maxPerMap:          3,
		maxMessageLength:   40,
		blockedMapPrefixes: []uint32{91, 92},
	}

	rm := Transform(m)

	got := Extract(rm)

	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}
