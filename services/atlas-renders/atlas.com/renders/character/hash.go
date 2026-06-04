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
// Ported from the former atlas-wz-extractor characterrender package (removed
// in the task-071 MinIO consolidation) so existing
// /api/wz/character/render/.../<hash>.png URLs continue to resolve after the
// atlas-renders cutover. LoadoutHash truncates to 16 hex chars; the atlas-ui
// producer (characterRender.service.ts) and the nginx character-render route
// (deploy/shared/routes.conf) must stay in lockstep with that length.
func CanonicalLoadoutString(
	tenant, region string,
	majorVersion, minorVersion uint16,
	skin, hair, face int,
	stance string,
	frame, resize int,
	items []int,
	gender int,
) string {
	sorted := append([]int(nil), items...)
	sort.Ints(sorted)
	parts := make([]string, len(sorted))
	for i, id := range sorted {
		parts[i] = strconv.Itoa(id)
	}
	return fmt.Sprintf(
		"%s|%s|%d.%d|%d|%d|%d|%s|%d|%d|%s|%d",
		tenant, region, majorVersion, minorVersion,
		skin, hair, face, stance, frame, resize,
		strings.Join(parts, ","),
		gender,
	)
}

// LoadoutHash returns the first 16 hex chars of SHA-256(canonical).
func LoadoutHash(canonical string) string {
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])[:16]
}
