package coupon

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	couponrules "github.com/Chronicle20/atlas/libs/atlas-constants/coupon"
)

// codeAlphabet omits O/0 and I/1/L: these codes are read off a screen and
// typed by hand, and an ambiguous character turns a valid code into a support
// ticket.
//
// ASCII ASSUMPTION. Every rune here is one byte, and couponrules.Plausible's
// length gate counts BYTES (len(code)) while the coupons.code column is
// varchar(32) and counts CHARACTERS. For a generated code the two agree
// exactly. For an admin-supplied code the byte count is >= the character
// count, so Plausible can only ever UNDER-accept a non-ASCII code — it never
// admits one that would overflow the column. That asymmetry is why the
// mismatch is safe and is recorded here rather than fixed.
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// DefaultGeneratedCodeLength is what POST /coupons uses when the admin leaves
// the code blank. Ten draws from a 31-character alphabet is log2(31)*10 ≈ 49.5
// bits, far past guessable, while still being short enough to read aloud.
const DefaultGeneratedCodeLength = 10

// blankCode reports whether an admin supplied no code at all — whitespace only
// counts as blank, since Normalize would trim it away to nothing anyway.
func blankCode(code string) bool {
	return couponrules.Normalize(code) == ""
}

// GenerateCode draws from crypto/rand. Coupon codes are SECRETS — a code drawn
// from math/rand is guessable from a handful of observed codes.
func GenerateCode(length int) (string, error) {
	if length <= 0 || length > couponrules.MaxCodeLength {
		return "", fmt.Errorf("%w: code length must be 1-%d", ErrInvalidCoupon, couponrules.MaxCodeLength)
	}
	out := make([]byte, length)
	limit := big.NewInt(int64(len(codeAlphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		out[i] = codeAlphabet[n.Int64()]
	}
	return string(out), nil
}

// IsDuplicateCode reports whether err is the (tenant_id, code) unique-index
// violation raised by CreateEntity.
//
// It matches TWO drivers on purpose. Production runs Postgres, which reports
// SQLSTATE 23505 through *pgconn.PgError — the same code
// redemption.IsUniqueViolation matches. The test harness runs gorm's SQLite
// driver (databasetest.NewInMemoryTenantDB), which reports
// "UNIQUE constraint failed: coupons.tenant_id, coupons.code" as a plain
// error carrying no SQLSTATE, so a Postgres-only matcher would make the 409
// mapping and the batch collision retry untestable on this branch's harness.
//
// gorm's TranslateError/ErrDuplicatedKey WOULD collapse both arms into a
// single errors.Is: gorm.io/driver/postgres@v1.6.2/error_translator.go:29-31
// returns fmt.Errorf("%w: %w", translatedErr, pgErr), so errors.As still
// resolves the *pgconn.PgError and redemption.IsUniqueViolation would keep
// working; and gorm.io/driver/sqlite@v1.6.0/error_translator.go:11-12 maps
// extended codes 1555/2067 to the same gorm.ErrDuplicatedKey. That
// simplification is SAFE and is not taken here only because of BLAST RADIUS:
// TranslateError is enabled on neither connection
// (libs/atlas-database/connection.go:125 opens with a bare &gorm.Config{}, as
// does databasetest), so turning it on changes the error surface of every
// service in the monorepo — a libs/atlas-database change, not a coupon one.
// If someone flips it repo-wide, replacing this function with a single
// errors.Is(err, gorm.ErrDuplicatedKey) is the correct follow-up.
func IsDuplicateCode(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return sqliteUniqueViolation(err)
}

// sqliteUniqueViolation matches gorm's SQLite driver, whose unique-constraint
// error carries no code to compare — only the message. Reached only under the
// test harness; Postgres never produces this text.
func sqliteUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
