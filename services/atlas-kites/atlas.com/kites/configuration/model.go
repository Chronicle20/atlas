package configuration

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// mapPrefixDivisor is the client's own field-id bucket size. The case-18 arm of
// CWvsContext::SendConsumeCashItemUseRequest (gms_v95 @0x9ed01e) divides
// GetCurFieldID() by 10,000,000 with the MSVC magic 0x6B5FCA6B / sar 22 and
// refuses when the quotient is 91 — the Free Market range. Atlas expresses that
// as a configurable prefix denylist rather than hard-coding one map id.
const mapPrefixDivisor = 10000000

// Model is the immutable per-tenant kite placement policy.
type Model struct {
	maxPerMap          int
	maxMessageLength   int
	blockedMapPrefixes []uint32
}

func (m Model) MaxPerMap() int {
	return m.maxPerMap
}

func (m Model) MaxMessageLength() int {
	return m.maxMessageLength
}

func (m Model) BlockedMapPrefixes() []uint32 {
	return m.blockedMapPrefixes
}

// IsMapBlocked reports whether the field's map id falls in a denied prefix
// bucket, mirroring the client's arithmetic.
func (m Model) IsMapBlocked(mapId _map.Id) bool {
	prefix := uint32(mapId) / mapPrefixDivisor
	for _, p := range m.blockedMapPrefixes {
		if p == prefix {
			return true
		}
	}
	return false
}

// DefaultConfig is used whenever a tenant has not provisioned kite-configs, and
// for any individual knob left at its zero value.
func DefaultConfig() Model {
	return Model{
		maxPerMap:        10,  // PRD FR-5.1
		maxMessageLength: 182, // 3 x 60-byte CUIHope edit controls + 2 '\n' joiners (CUIHope::OnCreate @0x7824f0)
		// 91 == the Free Market range (910000000-919999999), the client's own ban.
		blockedMapPrefixes: []uint32{91},
	}
}
