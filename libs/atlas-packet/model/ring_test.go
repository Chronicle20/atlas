package model

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// fixturePartnerSN is the two's-complement int64 of 0x99AABBCCDDEEFF00. It is
// computed at runtime (not as a constant conversion) because the literal
// exceeds int64's range as an untyped constant.
var fixturePartnerSN = func() int64 {
	u := uint64(0x99AABBCCDDEEFF00)
	return int64(u)
}()

// fixture values shared by every case (task-2 brief).
var (
	fixtureCouple = PairRing{
		OwnSN:     0x1122334455667788,
		PartnerSN: fixturePartnerSN,
		ItemId:    0x00001234,
	}
	fixtureFriendship = PairRing{
		OwnSN:     0x0102030405060708,
		PartnerSN: 0x1112131415161718,
		ItemId:    0x00005678,
	}
	fixtureMarriage = MarriageRing{
		MarriageCharacterId: 0x000000AA,
		PartnerCharacterId:  0x000000BB,
		ItemId:              0x0000CCDD,
	}
)

// hexBytes decodes a whitespace-separated hex fixture from the brief's table
// into raw bytes. It is the source of truth for expected wire bytes; total
// length is derived from it, never hardcoded separately.
func hexBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		t.Fatalf("bad hex fixture %q: %v", s, err)
	}
	return b
}

func ctxTenant(t *testing.T, region string, major, minor uint16) tenant.Model {
	t.Helper()
	ctx := test.CreateContext(region, major, minor)
	return tenant.MustFromContext(ctx)
}

// TestRingSetEncodeField pins the wire shape of RingSet.EncodeField against
// the task-2 brief's byte fixtures for GMS and JMS, including the empty-path
// invariant (PRD FR-9) and the GMS version-stability claim (v48/v83/v95
// byte-identical for the same input).
func TestRingSetEncodeField(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	encode := func(rs RingSet, tn tenant.Model) []byte {
		w := response.NewWriter(l)
		rs.EncodeField(w, tn)
		return w.Bytes()
	}

	cases := []struct {
		name     string
		region   string
		major    uint16
		rs       RingSet
		expected string
	}{
		{"GMS empty", "GMS", 83, RingSet{}, "00 00 00"},
		{"GMS couple only", "GMS", 83, RingSet{Couple: &fixtureCouple},
			"01 8877665544332211 00FFEEDDCCBBAA99 34120000 00 00"},
		{"GMS friendship only", "GMS", 83, RingSet{Friendship: &fixtureFriendship},
			"00 01 0807060504030201 1817161514131211 78560000 00"},
		{"GMS marriage only", "GMS", 83, RingSet{Marriage: &fixtureMarriage},
			"00 00 01 AA000000 BB000000 DDCC0000"},
		{"GMS all three", "GMS", 83, RingSet{Couple: &fixtureCouple, Friendship: &fixtureFriendship, Marriage: &fixtureMarriage},
			"01 8877665544332211 00FFEEDDCCBBAA99 34120000" +
				"01 0807060504030201 1817161514131211 78560000" +
				"01 AA000000 BB000000 DDCC0000"},
		{"GMS v48 empty", "GMS", 48, RingSet{}, "00 00 00"},
		{"GMS v95 all three", "GMS", 95, RingSet{Couple: &fixtureCouple, Friendship: &fixtureFriendship, Marriage: &fixtureMarriage},
			"01 8877665544332211 00FFEEDDCCBBAA99 34120000" +
				"01 0807060504030201 1817161514131211 78560000" +
				"01 AA000000 BB000000 DDCC0000"},
		{"JMS empty", "JMS", 185, RingSet{}, "00 00 00"},
		{"JMS couple only", "JMS", 185, RingSet{Couple: &fixtureCouple},
			"01 01000000 8877665544332211 00FFEEDDCCBBAA99 34120000 00 00"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tn := ctxTenant(t, c.region, c.major, 1)
			got := encode(c.rs, tn)
			want := hexBytes(t, c.expected)
			if !bytes.Equal(got, want) {
				t.Fatalf("got % x, want % x", got, want)
			}
		})
	}

	// GMS is version-stable v48..v95 for the same non-empty input (design.md
	// §2): confirm the all-three encode is byte-identical across the trio.
	rs := RingSet{Couple: &fixtureCouple, Friendship: &fixtureFriendship, Marriage: &fixtureMarriage}
	v48 := encode(rs, ctxTenant(t, "GMS", 48, 1))
	v83 := encode(rs, ctxTenant(t, "GMS", 83, 1))
	v95 := encode(rs, ctxTenant(t, "GMS", 95, 1))
	if !bytes.Equal(v48, v83) {
		t.Fatalf("GMS v48 all-three = % x, want byte-identical to v83 = % x", v48, v83)
	}
	if !bytes.Equal(v95, v83) {
		t.Fatalf("GMS v95 all-three = % x, want byte-identical to v83 = % x", v95, v83)
	}
}

// TestRingSetFieldRoundTrip verifies EncodeField/DecodeField agree
// field-by-field across every tenant variant and every populated combination,
// including that a nil arm decodes back to nil.
func TestRingSetFieldRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	combos := []struct {
		name string
		rs   RingSet
	}{
		{"empty", RingSet{}},
		{"couple only", RingSet{Couple: &fixtureCouple}},
		{"friendship only", RingSet{Friendship: &fixtureFriendship}},
		{"marriage only", RingSet{Marriage: &fixtureMarriage}},
		{"all three", RingSet{Couple: &fixtureCouple, Friendship: &fixtureFriendship, Marriage: &fixtureMarriage}},
	}

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			tn := ctxTenant(t, v.Region, v.MajorVersion, v.MinorVersion)
			for _, c := range combos {
				t.Run(c.name, func(t *testing.T) {
					w := response.NewWriter(l)
					c.rs.EncodeField(w, tn)
					encoded := w.Bytes()

					req := request.Request(encoded)
					reader := request.NewRequestReader(&req, 0)

					var got RingSet
					got.DecodeField(&reader, tn)

					if reader.Available() > 0 {
						t.Errorf("reader has %d unconsumed bytes after decode", reader.Available())
					}

					assertPairEqual(t, "Couple", c.rs.Couple, got.Couple)
					assertPairEqual(t, "Friendship", c.rs.Friendship, got.Friendship)

					if (c.rs.Marriage == nil) != (got.Marriage == nil) {
						t.Fatalf("Marriage nil-ness mismatch: want %v, got %v", c.rs.Marriage, got.Marriage)
					}
					if c.rs.Marriage != nil {
						if *c.rs.Marriage != *got.Marriage {
							t.Fatalf("Marriage mismatch: want %+v, got %+v", *c.rs.Marriage, *got.Marriage)
						}
					}
				})
			}
		})
	}
}

func assertPairEqual(t *testing.T, name string, want, got *PairRing) {
	t.Helper()
	if (want == nil) != (got == nil) {
		t.Fatalf("%s nil-ness mismatch: want %v, got %v", name, want, got)
	}
	if want != nil && *want != *got {
		t.Fatalf("%s mismatch: want %+v, got %+v", name, *want, *got)
	}
}
