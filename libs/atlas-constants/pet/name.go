// Package pet holds Atlas-canonical pet constants shared across services.
package pet

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// MinNameLength / MaxNameLength are the bounds the GMS v83 client's own pet-name
// input dialog enforces: sub_9AC7CB(dlg, NULL, 4, 12, 0, 1) @0xa0bb2f, where a3
// is the minimum and a4 the maximum. The (min,max) reading is fixed by three
// sibling call sites in the same binary — CTabParty::OnInvite @0x90e1da passes
// (4,12) for a character name, ask_SPW @0x9ad030 passes (8,8) for the 8-digit
// second password, ask_guildname @0x9ad131 passes (4,12) for a guild name.
//
// The pets.name column is size:13 (services/atlas-pets/.../pet/entity.go) — one
// wider than anything the client can send, deliberately. The column is NOT the
// binding constraint; these are.
const (
	MinNameLength = 4
	MaxNameLength = 12
)

var (
	ErrNameTooShort = errors.New("pet name is too short")
	ErrNameTooLong  = errors.New("pet name is too long")
)

// NormalizeName trims the surrounding whitespace the client's own
// TrimLeft/TrimRight removes before it encodes, so both sides validate the same
// string. Callers MUST normalize before calling ValidateName.
func NormalizeName(s string) string {
	return strings.TrimSpace(s)
}

// ValidateName reports whether an ALREADY-NORMALIZED name is acceptable. Length
// is counted in runes, not bytes: the bound the client enforces is a character
// count on its edit control, and a byte count would reject legal multi-byte
// names one character short.
func ValidateName(s string) error {
	n := utf8.RuneCountInString(s)
	if n < MinNameLength {
		return ErrNameTooShort
	}
	if n > MaxNameLength {
		return ErrNameTooLong
	}
	return nil
}
