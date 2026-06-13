package matrix

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Tiers holds the tier-1 membership definition from tiers.yaml.
// Three expansion granularities: explicit packets, packet_prefixes
// (dispatcher families by package dir), and opaque_types (Go type names;
// expansion via TypeRegistry recursion joins in Task 3.2).
type Tiers struct {
	OpaqueTypes    []string `yaml:"opaque_types"`
	PacketPrefixes []string `yaml:"packet_prefixes"`
	Packets        []string `yaml:"packets"`
}

// LoadTiers reads and parses tiers.yaml at path. A missing file is not an
// error — it returns empty Tiers (all packets tier 0), because not every
// invocation will have an evidence dir set up yet.
func LoadTiers(path string) (Tiers, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Tiers{}, nil
	}
	if err != nil {
		return Tiers{}, err
	}
	var t Tiers
	if err := yaml.Unmarshal(raw, &t); err != nil {
		return Tiers{}, err
	}
	return t, nil
}

// IsTier1 reports whether a packet id is tier-1. recurseTypes is the
// packet's transitive sub-struct type set (from atlaspacket.TypeRegistry);
// pass nil when unavailable (TypeRegistry wiring is Task 3.2).
func (t Tiers) IsTier1(packet string, recurseTypes []string) bool {
	for _, p := range t.Packets {
		if p == packet {
			return true
		}
	}
	for _, pre := range t.PacketPrefixes {
		if strings.HasPrefix(packet, pre) {
			return true
		}
	}
	for _, rt := range recurseTypes {
		for _, ot := range t.OpaqueTypes {
			if rt == ot {
				return true
			}
		}
	}
	return false
}
