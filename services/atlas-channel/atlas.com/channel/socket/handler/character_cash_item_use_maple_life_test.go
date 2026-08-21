package handler

import (
	"atlas-channel/maplelife"
	"atlas-channel/session"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// mapleLifeNoopWP is a writer.Producer that discards its output, the same
// shape as remoteMerchantNoopWP -- reused here rather than duplicated across
// two neighbouring test files at the reviewer's discretion, but this arm
// exercises a different set of handlers so it gets its own trivial copy.
func mapleLifeNoopWP(_ string) (swriter.BodyFunc, error) {
	return func(_ logrus.FieldLogger, _ context.Context) func(encoder packet.Encode) []byte {
		return func(_ packet.Encode) []byte { return nil }
	}, nil
}

// newMapleLifeArmTestSession is newCashItemUseTestSessionForVersion plus an
// AccountId stamp (via Processor.Create, see maple_life_open_test.go), so the
// Maple Life row can assert against the registry's account-keyed entry.
func newMapleLifeArmTestSession(t *testing.T, region string, major uint16, characterId uint32, accountId uint32) (session.Model, context.Context, tenant.Model, func()) {
	t.Helper()
	ten := mustTenant(t, region, major, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	sp := session.NewProcessor(logrus.New(), ctx)
	ch := channel.NewModel(world.Id(1), channel.Id(0))
	sp.Create(ch, 0)(sessionId, discardConn{})
	sp.SetCharacterId(sessionId, characterId)
	sp.SetAccountId(sessionId, accountId)
	f := field.NewBuilder(world.Id(1), channel.Id(0), _map.Id(100000000)).Build()
	updated := sp.SetField(sessionId, f)

	return updated, ctx, ten, func() { session.ClearRegistryForTenant(ten.Id()) }
}

// cashItemUsePrefixForVersion encodes the common cashsb.ItemUse prefix for
// the given tenant's UpdateTimeFirst shape (character_cash_item_use.go:42):
// a leading uint32 updateTime on GMS v87+/JMS, then int16 source, uint32
// itemId -- mirrors cashItemUsePrefix, generalised to the v95 leading-header
// shape the existing helper (v83-only) does not cover.
func cashItemUsePrefixForVersion(updateTimeFirst bool, updateTime uint32, slot int16, itemId uint32) []byte {
	var raw []byte
	if updateTimeFirst {
		raw = append(raw, byte(updateTime), byte(updateTime>>8), byte(updateTime>>16), byte(updateTime>>24))
	}
	raw = append(raw, byte(slot), byte(slot>>8))
	raw = append(raw, byte(itemId), byte(itemId>>8), byte(itemId>>16), byte(itemId>>24))
	return raw
}

func TestCashItemNeighbourArmsUnaffectedByMapleLife(t *testing.T) {
	const testSlot = int16(3)

	type tc struct {
		name          string
		itemId        uint32
		wantTypeV83   CashSlotItemType
		wantTypeV95   CashSlotItemType
		wantMapleLife bool
	}
	cases := []tc{
		{name: "pet multi-consumable", itemId: 5460000, wantTypeV83: CashSlotItemType(57), wantTypeV95: CashSlotItemType(58), wantMapleLife: false},
		{name: "sealing lock (timed)", itemId: 5061000, wantTypeV83: CashSlotItemType(64), wantTypeV95: CashSlotItemType(65), wantMapleLife: false},
		{name: "vicious hammer", itemId: 5570000, wantTypeV83: CashSlotItemType(66), wantTypeV95: CashSlotItemType(67), wantMapleLife: false},
		{name: "maple life", itemId: 5431000, wantTypeV83: CashSlotItemType(65), wantTypeV95: CashSlotItemType(66), wantMapleLife: true},
	}

	for _, c := range cases {
		for _, v := range []struct {
			name     string
			major    uint16
			wantType func(tc) CashSlotItemType
		}{
			{name: "v83", major: 83, wantType: func(c tc) CashSlotItemType { return c.wantTypeV83 }},
			{name: "v95", major: 95, wantType: func(c tc) CashSlotItemType { return c.wantTypeV95 }},
		} {
			t.Run(c.name+"/"+v.name, func(t *testing.T) {
				ten := mustTenant(t, "GMS", v.major, 1)

				// Structural check: GetCashSlotItemType really does collide
				// the way the PRD says it does -- this is what makes the
				// Maple Life row meaningful (task-246 brief).
				if got := GetCashSlotItemType(ten)(item.Id(c.itemId)); got != v.wantType(c) {
					t.Fatalf("GetCashSlotItemType = %d, want %d", got, v.wantType(c))
				}

				restoreSlot := installCashItemInSlotSeam(t, testSlot, c.itemId)
				defer restoreSlot()

				const accountId = uint32(900)
				const characterId = uint32(901)
				s, ctx, sessTen, cleanup := newMapleLifeArmTestSession(t, "GMS", v.major, characterId, accountId)
				defer cleanup()

				updateTimeFirst := v.major >= 87
				raw := cashItemUsePrefixForVersion(updateTimeFirst, 42, testSlot, c.itemId)
				switch c.name {
				case "maple life":
					// ItemUseMapleLife sub-body: name, al[0..3], gender,
					// currentClass, sp, trailing update_time (always written
					// regardless of updateTimeFirst -- see
					// cash/serverbound/item_use_maple_life.go).
					raw = append(raw, encodeAsciiString("Name")...)
					raw = append(raw, littleEndianInt32(0)...)  // al0
					raw = append(raw, littleEndianInt32(0)...)  // al1
					raw = append(raw, littleEndianInt32(0)...)  // al2
					raw = append(raw, littleEndianInt32(0)...)  // al3
					raw = append(raw, littleEndianInt32(0)...)  // gender
					raw = append(raw, littleEndianInt32(0)...)  // currentClass
					raw = append(raw, littleEndianInt32(0)...)  // sp
					raw = append(raw, littleEndianInt32(99)...) // trailing updateTime
				case "sealing lock (timed)":
					// ItemUseSeal sub-body: inventoryType, slot, trailing
					// updateTime iff !updateTimeFirst.
					raw = append(raw, littleEndianInt32(1)...) // inventoryType (EQUIP)
					raw = append(raw, littleEndianInt32(int32(testSlot))...)
					if !updateTimeFirst {
						raw = append(raw, littleEndianInt32(99)...)
					}
				case "vicious hammer":
					// ItemUseViciousHammer sub-body: itemTI, slotPosition,
					// updateTime (unconditional, every version).
					raw = append(raw, littleEndianInt32(int32(c.itemId))...)
					raw = append(raw, littleEndianInt32(int32(testSlot))...)
					raw = append(raw, littleEndianInt32(99)...)
				default:
					// pet multi-consumable has no dedicated arm at all --
					// falls through to the handler's default warn with no
					// further bytes to decode.
				}

				req := request.Request(raw)
				reader := request.NewRequestReader(&req, 0)

				handlerFunc := CharacterCashItemUseHandleFunc(logrus.New(), ctx, mapleLifeNoopWP)
				handlerFunc(s, &reader, map[string]interface{}{})

				_, ok := maplelife.GetRegistry().Get(sessTen, accountId)
				if ok != c.wantMapleLife {
					t.Fatalf("maplelife.Get() ok = %v, want %v (arm reached should%s be Maple Life)", ok, c.wantMapleLife, negate(c.wantMapleLife))
				}
			})
		}
	}
}

func negate(want bool) string {
	if want {
		return ""
	}
	return " not"
}

func encodeAsciiString(s string) []byte {
	n := int16(len(s))
	out := []byte{byte(n), byte(n >> 8)}
	out = append(out, []byte(s)...)
	return out
}

func littleEndianInt32(v int32) []byte {
	u := uint32(v)
	return []byte{byte(u), byte(u >> 8), byte(u >> 16), byte(u >> 24)}
}

// TestMapleLifeArmUsesNoBareCashSlotTypeComparison asserts, by reading the
// handler's own source, that the classification-543 arm never falls back to
// comparing `it` against the colliding CashSlotItemType constants (PRD
// FR-2.2). This makes the ban executable: a future edit that "simplifies"
// the arm back to a type comparison fails this test even if every other
// test still happens to pass by coincidence of test-data choice.
func TestMapleLifeArmUsesNoBareCashSlotTypeComparison(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	src, err := os.ReadFile(filepath.Join(wd, "character_cash_item_use.go"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	body := string(src)
	start := strings.Index(body, "category == item.ClassificationCharacterCreation")
	if start == -1 {
		t.Fatal("could not locate the Maple Life arm (category == item.ClassificationCharacterCreation)")
	}
	end := strings.Index(body[start:], "\n\t\t}\n\n\t\tif category == item.ClassificationMegaphones")
	arm := body[start:]
	if end != -1 {
		arm = body[start : start+end]
	} else {
		// Fall back to a fixed window past the arm if the next classification
		// arm's text ever changes shape -- still far larger than the arm itself.
		if len(arm) > 1200 {
			arm = arm[:1200]
		}
	}
	for _, forbidden := range []string{"CashSlotItemType(57)", "CashSlotItemType(58)", "CashSlotItemType(65)", "CashSlotItemType(66)"} {
		if strings.Contains(arm, forbidden) {
			t.Errorf("Maple Life arm source contains forbidden bare comparison %q (PRD FR-2.2)", forbidden)
		}
	}
}

func TestMapleLifeSupported(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"GMS 83", true},
		{"GMS 84", false}, // task-246 Task 2 ruling: CUICharacterSaleDlg is VERSION-ABSENT on gms_v84.
		{"GMS 87", true},
		{"GMS 92", true},
		{"GMS 95", true},
		{"GMS 79", false},
		{"GMS 72", false},
		{"GMS 61", false},
		{"GMS 48", false},
		{"JMS 185", false},
	}
	versions := map[string]struct {
		region string
		major  uint16
	}{
		"GMS 83": {"GMS", 83}, "GMS 84": {"GMS", 84}, "GMS 87": {"GMS", 87}, "GMS 92": {"GMS", 92}, "GMS 95": {"GMS", 95},
		"GMS 79": {"GMS", 79}, "GMS 72": {"GMS", 72}, "GMS 61": {"GMS", 61}, "GMS 48": {"GMS", 48}, "JMS 185": {"JMS", 185},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := versions[c.name]
			ten := mustTenant(t, v.region, v.major, 1)
			if got := mapleLifeSupported(ten); got != c.want {
				t.Errorf("mapleLifeSupported(%s v%d) = %v, want %v", v.region, v.major, got, c.want)
			}
		})
	}
}

func TestMapleLifeUnsupportedVersionWritesNothing(t *testing.T) {
	for _, v := range []struct {
		name  string
		major uint16
	}{
		{"v79", 79},
		{"v84", 84}, // the non-obvious exclusion (task-246 Task 2 ruling).
	} {
		t.Run(v.name, func(t *testing.T) {
			const testSlot = int16(3)
			const itemId = uint32(5431000)
			restoreSlot := installCashItemInSlotSeam(t, testSlot, itemId)
			defer restoreSlot()

			const accountId = uint32(910)
			const characterId = uint32(911)
			s, ctx, ten, cleanup := newMapleLifeArmTestSession(t, "GMS", v.major, characterId, accountId)
			defer cleanup()

			calls := &wpCallCounter{}
			// updateTimeFirst is false for both v79 and v84 (GMS < 87), so
			// the common prefix is exactly 6 bytes: int16 source + uint32
			// itemId (cash/serverbound/item_use.go). Nothing beyond it
			// belongs to this arm at all on an unsupported version.
			const commonPrefixLen = 6
			raw := cashItemUsePrefixForVersion(false, 42, testSlot, itemId)
			if len(raw) != commonPrefixLen {
				t.Fatalf("test fixture prefix length = %d, want %d", len(raw), commonPrefixLen)
			}
			req := request.Request(raw)
			reader := request.NewRequestReader(&req, 0)

			handlerFunc := CharacterCashItemUseHandleFunc(logrus.New(), ctx, calls.producer)
			handlerFunc(s, &reader, map[string]interface{}{})

			if calls.n != 0 {
				t.Errorf("writer.Producer called %d times, want 0", calls.n)
			}
			if _, ok := maplelife.GetRegistry().Get(ten, accountId); ok {
				t.Errorf("expected no maplelife registry entry on an unsupported version")
			}
			// FR-2.4's third half: the sub-body was never decoded. mapleLifeSupported
			// gates BEFORE cashsb.NewItemUseMapleLife(...).Decode(...) is ever reached
			// (character_cash_item_use.go, the `if !mapleLifeSupported(t) { ...; return }`
			// guard precedes the Decode call), so the reader must be left exactly where
			// the common ItemUse prefix decode left it -- nothing past commonPrefixLen
			// consumed.
			if got := reader.Position(); got != commonPrefixLen {
				t.Errorf("reader.Position() = %d, want %d (sub-body must be left unconsumed)", got, commonPrefixLen)
			}
		})
	}
}

type wpCallCounter struct{ n int }

func (c *wpCallCounter) producer(name string) (swriter.BodyFunc, error) {
	c.n++
	return func(_ logrus.FieldLogger, _ context.Context) func(encoder packet.Encode) []byte {
		return func(_ packet.Encode) []byte { return nil }
	}, nil
}
