package character

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// CanonicalLoadoutString returns the canonical input string used to derive a
// loadout hash. Equipment ids are sorted ascending so that input order does
// not affect the hash.
//
// Ported verbatim from
// services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/hash.go
// so existing /api/wz/character/render/.../<hash>.png URLs continue to
// resolve after the atlas-renders cutover.
func CanonicalLoadoutString(
	tenant, region string,
	majorVersion, minorVersion uint16,
	skin, hair, face int,
	stance string,
	frame, resize int,
	items []int,
) string {
	sorted := append([]int(nil), items...)
	sort.Ints(sorted)
	parts := make([]string, len(sorted))
	for i, id := range sorted {
		parts[i] = strconv.Itoa(id)
	}
	return fmt.Sprintf(
		"%s|%s|%d.%d|%d|%d|%d|%s|%d|%d|%s",
		tenant, region, majorVersion, minorVersion,
		skin, hair, face, stance, frame, resize,
		strings.Join(parts, ","),
	)
}

// LoadoutHash returns the first 16 hex chars of SHA-256(canonical).
func LoadoutHash(canonical string) string {
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])[:16]
}
