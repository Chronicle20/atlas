// Package coupon holds the canonical cash-shop coupon code normalization and
// plausibility rules shared by atlas-channel and atlas-cashshop.
package coupon

import "strings"

// MaxCodeLength matches the coupons.code column width. A submission longer
// than this cannot match any stored code, so it is rejected without a query.
//
// This constant must stay in step with the coupons.code column width
// introduced by the migration that creates the coupons table.
const MaxCodeLength = 32

// Normalize is the ONE canonical form a coupon code takes: surrounding
// whitespace trimmed, then uppercased. Codes are STORED normalized, so the
// unique index on (tenant_id, code) is what makes lookups case-insensitive —
// there is no functional index over a raw value.
//
// Interior whitespace is deliberately preserved: it is part of the code.
func Normalize(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// Plausible reports whether an ALREADY-NORMALIZED code could possibly match a
// stored coupon. It is a cheap local gate — the first line of brute-force
// defence — not a validity check. It rejects empty, over-length, and
// anything that is not already its own normal form.
func Plausible(code string) bool {
	if code == "" || len(code) > MaxCodeLength {
		return false
	}
	return code == Normalize(code)
}
