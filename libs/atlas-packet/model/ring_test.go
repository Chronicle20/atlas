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

// record-block fixtures (task-3b brief). Field values are chosen to make
// each byte distinguishable in the fixture hex, not to model realistic game
// data.
var (
	fixtureCoupleRecord = CoupleRecord{
		PairCharacterId:   0x00112233,
		PairCharacterName: "Ann",
		OwnSN:             0x0102030405060708,
		PairSN:            0x1112131415161718,
	}
	fixtureFriendRecord = FriendRecord{
		CoupleRecord: fixtureCoupleRecord,
		FriendItemId: 0x00005678,
	}
	fixtureMarriageRecord = MarriageRecord{
		MarriageNo:  1,
		GroomId:     0x000000AA,
		BrideId:     0x000000BB,
		Status:      2,
		GroomItemId: 0x0000CCDD,
		BrideItemId: 0x0000EEFF,
		GroomName:   "Sam",
		BrideName:   "Kay",
	}
)

// coupleRecordHex/friendRecordHex/marriageRecordHex are the byte-by-byte wire
// encodings of the fixtures above, derived field-by-field (little-endian ints,
// zero-padded 13-byte names) per the task-3b brief's pinned layouts.
const (
	coupleRecordHex = "33 22 11 00" + // PairCharacterId
		" 41 6E 6E 00 00 00 00 00 00 00 00 00 00" + // PairCharacterName "Ann", zero-padded to 13
		" 08 07 06 05 04 03 02 01" + // OwnSN
		" 18 17 16 15 14 13 12 11" // PairSN

	friendRecordHex = coupleRecordHex +
		" 78 56 00 00" // FriendItemId

	marriageRecordHex = "01 00 00 00" + // MarriageNo
		" AA 00 00 00" + // GroomId
		" BB 00 00 00" + // BrideId
		" 02 00" + // Status
		" DD CC 00 00" + // GroomItemId
		" FF EE 00 00" + // BrideItemId
		" 53 61 6D 00 00 00 00 00 00 00 00 00 00" + // GroomName "Sam", zero-padded to 13
		" 4B 61 79 00 00 00 00 00 00 00 00 00 00" // BrideName "Kay", zero-padded to 13
)

// TestRingRecordsEncode pins the wire shape of RingRecords.EncodeRecords
// against the task-3b brief's byte fixtures, including the empty-path
// invariant (PRD FR-9) on modern GMS/JMS and legacy GMS (<=28), and the
// 12-byte name-truncation rule (task-269 3a derivation).
func TestRingRecordsEncode(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	encode := func(rr RingRecords, tn tenant.Model) []byte {
		w := response.NewWriter(l)
		rr.EncodeRecords(w, tn)
		return w.Bytes()
	}

	cases := []struct {
		name     string
		region   string
		major    uint16
		rr       RingRecords
		expected string
	}{
		{"empty, modern", "GMS", 83, RingRecords{}, "00 00 00 00 00 00"},
		{"empty, legacy", "GMS", 28, RingRecords{}, "00 00"},
		{"empty, JMS", "JMS", 185, RingRecords{}, "00 00 00 00 00 00"},
		{"one couple record", "GMS", 83,
			RingRecords{Couple: []CoupleRecord{fixtureCoupleRecord}},
			"01 00" + coupleRecordHex + "00 00" + "00 00"},
		{"one of each", "GMS", 83,
			RingRecords{
				Couple:   []CoupleRecord{fixtureCoupleRecord},
				Friend:   []FriendRecord{fixtureFriendRecord},
				Marriage: []MarriageRecord{fixtureMarriageRecord},
			},
			"01 00" + coupleRecordHex + "01 00" + friendRecordHex + "01 00" + marriageRecordHex},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tn := ctxTenant(t, c.region, c.major, 1)
			got := encode(c.rr, tn)
			want := hexBytes(t, c.expected)
			if !bytes.Equal(got, want) {
				t.Fatalf("got % x, want % x", got, want)
			}
		})
	}

	// recordNameBytes extracts the 13-byte PairCharacterName field (record
	// offset 0x04..0x10) from an encoding of a single couple record: 2 bytes
	// of count prefix, then 4 bytes of PairCharacterId, then the name.
	recordNameBytes := func(encoded []byte) []byte {
		return encoded[2+4 : 2+4+recordNameWidth]
	}

	t.Run("name truncation", func(t *testing.T) {
		tn := ctxTenant(t, "GMS", 83, 1)
		rr := RingRecords{Couple: []CoupleRecord{{PairCharacterName: "ABCDEFGHIJKLMNOPQRST"}}} // 20 chars
		got := recordNameBytes(encode(rr, tn))
		want := hexBytes(t, "41 42 43 44 45 46 47 48 49 4A 4B 4C 00") // first 12 bytes + one 00
		if !bytes.Equal(got, want) {
			t.Fatalf("got % x, want % x", got, want)
		}
	})

	t.Run("name padding", func(t *testing.T) {
		tn := ctxTenant(t, "GMS", 83, 1)
		rr := RingRecords{Couple: []CoupleRecord{{PairCharacterName: "Ab"}}}
		got := recordNameBytes(encode(rr, tn))
		want := hexBytes(t, "41 62 00 00 00 00 00 00 00 00 00 00 00") // "Ab" + eleven 00
		if !bytes.Equal(got, want) {
			t.Fatalf("got % x, want % x", got, want)
		}
	})
}

// TestRingRecordsRoundTrip verifies EncodeRecords/DecodeRecords agree
// record-by-record across every tenant variant and every populated
// combination, including the legacy-GMS gate that omits the friend/marriage
// arms entirely.
func TestRingRecordsRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	combos := []struct {
		name string
		rr   RingRecords
	}{
		{"empty", RingRecords{}},
		{"couple only", RingRecords{Couple: []CoupleRecord{fixtureCoupleRecord}}},
		{"friend only", RingRecords{Friend: []FriendRecord{fixtureFriendRecord}}},
		{"marriage only", RingRecords{Marriage: []MarriageRecord{fixtureMarriageRecord}}},
		{"one of each", RingRecords{
			Couple:   []CoupleRecord{fixtureCoupleRecord},
			Friend:   []FriendRecord{fixtureFriendRecord},
			Marriage: []MarriageRecord{fixtureMarriageRecord},
		}},
	}

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			tn := ctxTenant(t, v.Region, v.MajorVersion, v.MinorVersion)
			for _, c := range combos {
				t.Run(c.name, func(t *testing.T) {
					w := response.NewWriter(l)
					c.rr.EncodeRecords(w, tn)
					encoded := w.Bytes()

					req := request.Request(encoded)
					reader := request.NewRequestReader(&req, 0)

					var got RingRecords
					got.DecodeRecords(&reader, tn)

					if reader.Available() > 0 {
						t.Errorf("reader has %d unconsumed bytes after decode", reader.Available())
					}

					wantCouple := c.rr.Couple
					wantFriend := c.rr.Friend
					wantMarriage := c.rr.Marriage
					legacy := v.Region == "GMS" && v.MajorVersion <= 28
					if legacy {
						wantFriend = nil
						wantMarriage = nil
					}

					assertCoupleRecordsEqual(t, wantCouple, got.Couple)
					assertFriendRecordsEqual(t, wantFriend, got.Friend)
					assertMarriageRecordsEqual(t, wantMarriage, got.Marriage)
				})
			}
		})
	}
}

func assertCoupleRecordsEqual(t *testing.T, want, got []CoupleRecord) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("Couple length mismatch: want %d, got %d", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("Couple[%d] mismatch: want %+v, got %+v", i, want[i], got[i])
		}
	}
}

func assertFriendRecordsEqual(t *testing.T, want, got []FriendRecord) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("Friend length mismatch: want %d, got %d", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("Friend[%d] mismatch: want %+v, got %+v", i, want[i], got[i])
		}
	}
}

func assertMarriageRecordsEqual(t *testing.T, want, got []MarriageRecord) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("Marriage length mismatch: want %d, got %d", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("Marriage[%d] mismatch: want %+v, got %+v", i, want[i], got[i])
		}
	}
}
