package handler

import (
	character2 "atlas-channel/character"
	"atlas-channel/character/factory"
	"atlas-channel/maplelife"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	mlcb "github.com/Chronicle20/atlas/libs/atlas-packet/maplelife/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	mapleLifeCreateTestAccountId   = uint32(7)
	mapleLifeCreateTestCharacterId = uint32(42)
	mapleLifeCreateTestWorldId     = world.Id(1)
	mapleLifeCreateTestItemId      = item.MapleLifeATypeId // ClassificationCharacterCreation (543)
	mapleLifeCreateTestSlot        = slot.Position(-3)
)

// Distinct non-real resolved bytes, same rationale as
// maple_life_check_name_test.go's mapleLifeCheckNameByte* constants (DOM-25
// proof: the handler must go through the config-resolved path).
const (
	mapleLifeCreateByteSuccess         = 0x60
	mapleLifeCreateByteNameTakenSubmit = 0x61
	mapleLifeCreateByteUnknown         = 0x62
)

// originalCreateMapleLifeFunc captures the real createMapleLifeFunc -- the
// one that runs the al0..al3/gender/currentClass/sp mapping (design.md §11
// A3) -- before newMapleLifeCreateEnv overrides the package var with a
// capturing stub. TestMapleLifeCreateMapsTheWireToTheFactoryRequest restores
// this so the mapping runs for real, and intercepts one level below it via
// newFactoryProcessorFunc instead.
var originalCreateMapleLifeFunc = createMapleLifeFunc

// fakeFactoryProcessor is a factory.Processor test double that records
// CreateMapleLife's arguments -- the mapped ints createMapleLifeFunc actually
// forwards -- rather than the raw wire sub the createMapleLifeFunc-level seam
// captures. SeedCharacter is unused by the Maple Life create path and panics
// if called.
type fakeFactoryProcessor struct {
	createMapleLifeCalls []struct {
		accountId    uint32
		worldId      world.Id
		name         string
		classOrdinal uint32
		gender       byte
		face         uint32
		hair         uint32
		hairColor    uint32
		skinColor    byte
		sp           byte
	}
	transactionId string
	err           error
}

func (f *fakeFactoryProcessor) SeedCharacter(uint32, world.Id, string, uint32, uint16, uint32, uint32, uint32, uint32, byte, uint32, uint32, uint32, uint32, byte, byte, byte, byte) (string, error) {
	panic("fakeFactoryProcessor.SeedCharacter: not used by the Maple Life create path")
}

func (f *fakeFactoryProcessor) CreateMapleLife(accountId uint32, worldId world.Id, name string, classOrdinal uint32, gender byte, face uint32, hair uint32, hairColor uint32, skinColor byte, sp byte) (string, error) {
	f.createMapleLifeCalls = append(f.createMapleLifeCalls, struct {
		accountId    uint32
		worldId      world.Id
		name         string
		classOrdinal uint32
		gender       byte
		face         uint32
		hair         uint32
		hairColor    uint32
		skinColor    byte
		sp           byte
	}{accountId: accountId, worldId: worldId, name: name, classOrdinal: classOrdinal, gender: gender, face: face, hair: hair, hairColor: hairColor, skinColor: skinColor, sp: sp})
	return f.transactionId, f.err
}

var _ factory.Processor = (*fakeFactoryProcessor)(nil)

type mapleLifeCreateEnv struct {
	t         *testing.T
	ctx       context.Context
	s         session.Model
	l         logrus.FieldLogger
	wp        writer.Producer
	announced []struct {
		writer string
		body   []byte
	}

	slotTemplateId uint32
	slotErr        error
	slotCalls      []struct {
		characterId uint32
		slot        int16
	}

	accountSlots     int16
	accountSlotErr   error
	accountSlotCalls []struct {
		accountId uint32
		worldId   world.Id
	}

	incrementSlotsResult int16
	incrementSlotsErr    error
	incrementSlotsCalls  []struct {
		accountId uint32
		worldId   world.Id
	}

	charactersInWorld    int
	charactersInWorldErr error

	validity    character2.NameValidityResult
	validityErr error
	nameCalls   []struct {
		name    string
		worldId world.Id
		scope   character2.NameScope
	}

	mapleLifeTransactionId string
	mapleLifeErr           error
	mapleLifeCalls         []struct {
		accountId uint32
		worldId   world.Id
		sub       cashsb.ItemUseMapleLife
	}
}

func newMapleLifeCreateEnv(t *testing.T) *mapleLifeCreateEnv {
	t.Helper()

	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	sp := session.NewProcessor(logrus.New(), ctx)
	ch := channel.NewModel(mapleLifeCreateTestWorldId, channel.Id(0))
	sp.Create(ch, 0)(sessionId, discardConn{})
	t.Cleanup(func() { session.ClearRegistryForTenant(ten.Id()) })

	sp.SetAccountId(sessionId, mapleLifeCreateTestAccountId)
	sp.SetCharacterId(sessionId, mapleLifeCreateTestCharacterId)
	f := field.NewBuilder(mapleLifeCreateTestWorldId, channel.Id(0), _map.Id(100000000)).Build()
	updated := sp.SetField(sessionId, f)

	t.Cleanup(func() { maplelife.GetRegistry().ClearAccount(ten, mapleLifeCreateTestAccountId) })

	env := &mapleLifeCreateEnv{t: t, ctx: ctx, s: updated, l: logrus.New()}
	// Defaults: every gate passes and the factory call succeeds.
	env.slotTemplateId = uint32(mapleLifeCreateTestItemId)
	env.accountSlots = 3
	env.charactersInWorld = 0
	env.validity = character2.NameValidityResult{Valid: true}
	env.mapleLifeTransactionId = "tx-1"

	env.wp = func(name string) (swriter.BodyFunc, error) {
		return func(bl logrus.FieldLogger, bctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(bl, bctx)(map[string]interface{}{
					"operations": map[string]interface{}{
						mlcb.MapleLifeErrorSuccess:           float64(mapleLifeCreateByteSuccess),
						mlcb.MapleLifeErrorNameTakenAtSubmit: float64(mapleLifeCreateByteNameTakenSubmit),
						mlcb.MapleLifeErrorUnknownError:      float64(mapleLifeCreateByteUnknown),
					},
				})
				env.announced = append(env.announced, struct {
					writer string
					body   []byte
				}{writer: name, body: b})
				return b
			}
		}, nil
	}

	origSlot := cashItemInSlotFunc
	cashItemInSlotFunc = func(_ logrus.FieldLogger, _ context.Context, characterId uint32, s int16) (uint32, error) {
		env.slotCalls = append(env.slotCalls, struct {
			characterId uint32
			slot        int16
		}{characterId: characterId, slot: s})
		return env.slotTemplateId, env.slotErr
	}
	t.Cleanup(func() { cashItemInSlotFunc = origSlot })

	origAccountSlots := accountSlotsFunc
	accountSlotsFunc = func(_ logrus.FieldLogger, _ context.Context, accountId uint32, worldId world.Id) (int16, error) {
		env.accountSlotCalls = append(env.accountSlotCalls, struct {
			accountId uint32
			worldId   world.Id
		}{accountId: accountId, worldId: worldId})
		return env.accountSlots, env.accountSlotErr
	}
	t.Cleanup(func() { accountSlotsFunc = origAccountSlots })

	origIncrementSlots := incrementAccountSlotsFunc
	incrementAccountSlotsFunc = func(_ logrus.FieldLogger, _ context.Context, accountId uint32, worldId world.Id) (int16, error) {
		env.incrementSlotsCalls = append(env.incrementSlotsCalls, struct {
			accountId uint32
			worldId   world.Id
		}{accountId: accountId, worldId: worldId})
		return env.incrementSlotsResult, env.incrementSlotsErr
	}
	t.Cleanup(func() { incrementAccountSlotsFunc = origIncrementSlots })

	origCharsInWorld := charactersInWorldFunc
	charactersInWorldFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32, _ world.Id) ([]character2.Model, error) {
		if env.charactersInWorldErr != nil {
			return nil, env.charactersInWorldErr
		}
		return make([]character2.Model, env.charactersInWorld), nil
	}
	t.Cleanup(func() { charactersInWorldFunc = origCharsInWorld })

	origValidity := mapleLifeNameValidityFunc
	mapleLifeNameValidityFunc = func(_ logrus.FieldLogger, _ context.Context, name string, worldId world.Id, scope character2.NameScope) (character2.NameValidityResult, error) {
		env.nameCalls = append(env.nameCalls, struct {
			name    string
			worldId world.Id
			scope   character2.NameScope
		}{name: name, worldId: worldId, scope: scope})
		return env.validity, env.validityErr
	}
	t.Cleanup(func() { mapleLifeNameValidityFunc = origValidity })

	origMapleLife := createMapleLifeFunc
	createMapleLifeFunc = func(_ logrus.FieldLogger, _ context.Context, accountId uint32, worldId world.Id, sub cashsb.ItemUseMapleLife) (string, error) {
		env.mapleLifeCalls = append(env.mapleLifeCalls, struct {
			accountId uint32
			worldId   world.Id
			sub       cashsb.ItemUseMapleLife
		}{accountId: accountId, worldId: worldId, sub: sub})
		return env.mapleLifeTransactionId, env.mapleLifeErr
	}
	t.Cleanup(func() { createMapleLifeFunc = origMapleLife })

	return env
}

func (e *mapleLifeCreateEnv) tenant() tenant.Model {
	return tenant.MustFromContext(e.ctx)
}

// mapleLifeSubmitParams is the full set of ItemUseMapleLife fields a submit
// sub-body carries (derivation.md §2: sName, al[0..3], nGender,
// nCurrentClass, nSP, update_time).
type mapleLifeSubmitParams struct {
	name         string
	al0          int32
	al1          int32
	al2          int32
	al3          int32
	gender       int32
	currentClass int32
	sp           int32
}

// mapleLifeSubmitSub builds a real decoded ItemUseMapleLife the same way the
// classification-543 arm would (character_cash_item_use.go), by round
// tripping through its own wire codec -- littleEndianInt32/encodeAsciiString
// are shared with character_cash_item_use_maple_life_test.go.
func mapleLifeSubmitSub(name string) cashsb.ItemUseMapleLife {
	return mapleLifeSubmitSubWith(mapleLifeSubmitParams{name: name})
}

// mapleLifeSubmitSubWith is mapleLifeSubmitSub's general form, for cases that
// need to control al0..al3/gender/currentClass/sp rather than accept the
// all-zero defaults.
func mapleLifeSubmitSubWith(p mapleLifeSubmitParams) cashsb.ItemUseMapleLife {
	raw := append([]byte{}, encodeAsciiString(p.name)...)
	raw = append(raw, littleEndianInt32(p.al0)...)
	raw = append(raw, littleEndianInt32(p.al1)...)
	raw = append(raw, littleEndianInt32(p.al2)...)
	raw = append(raw, littleEndianInt32(p.al3)...)
	raw = append(raw, littleEndianInt32(p.gender)...)
	raw = append(raw, littleEndianInt32(p.currentClass)...)
	raw = append(raw, littleEndianInt32(p.sp)...)
	raw = append(raw, littleEndianInt32(1)...) // updateTime

	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)
	sp := cashsb.NewItemUseMapleLife(false)
	sp.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})
	return *sp
}

func (e *mapleLifeCreateEnv) dispatch(sub cashsb.ItemUseMapleLife) {
	e.t.Helper()
	e.dispatchItem(mapleLifeCreateTestItemId, sub)
}

// dispatchItem is dispatch's general form for the B-Type coverage below,
// which must submit a different itemId than the suite's A-Type default.
// Callers are responsible for setting env.slotTemplateId to match, so gate
// 2's ownership re-check (templateId == itemId) still passes.
func (e *mapleLifeCreateEnv) dispatchItem(itemId item.Id, sub cashsb.ItemUseMapleLife) {
	e.t.Helper()
	handleMapleLifeCreate(e.l, e.ctx, e.wp)(e.s, itemId, mapleLifeCreateTestSlot, sub)
}

// lastArm decodes the MAPLELIFE_ERROR body the handler wrote (nType, nParam)
// and returns the resolved arm byte, or 0 with ok=false if nothing was
// announced.
func (e *mapleLifeCreateEnv) lastArm() (byte, bool) {
	if len(e.announced) == 0 {
		return 0, false
	}
	a := e.announced[len(e.announced)-1]
	if a.writer != mlcb.MapleLifeErrorWriter {
		e.t.Fatalf("wrote [%s], want [%s]", a.writer, mlcb.MapleLifeErrorWriter)
	}
	if len(a.body) < 1 {
		e.t.Fatalf("body too short: %x", a.body)
	}
	return a.body[0], true
}

func TestMapleLifeCreatePreCheckOrder(t *testing.T) {
	cases := []struct {
		name           string
		setup          func(e *mapleLifeCreateEnv)
		wantArm        byte
		wantNoAnnounce bool
	}{
		{
			name: "ownership lost",
			setup: func(e *mapleLifeCreateEnv) {
				e.slotTemplateId = uint32(mapleLifeCreateTestItemId) + 1
			},
			wantArm: mapleLifeCreateByteUnknown,
		},
		{
			name: "ownership wrong classification",
			setup: func(e *mapleLifeCreateEnv) {
				// 4000000 is a plain equip template, not classification 543.
				e.slotTemplateId = 4000000
			},
			wantArm: mapleLifeCreateByteUnknown,
		},
		{
			name: "slot limit reached",
			setup: func(e *mapleLifeCreateEnv) {
				e.accountSlots = 3
				e.charactersInWorld = 3
			},
			wantArm: mapleLifeCreateByteUnknown,
		},
		{
			name: "name taken at submit",
			setup: func(e *mapleLifeCreateEnv) {
				e.validity = character2.NameValidityResult{Valid: false, Reason: "duplicate"}
			},
			wantArm: mapleLifeCreateByteNameTakenSubmit,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newMapleLifeCreateEnv(t)
			tc.setup(e)

			e.dispatch(mapleLifeSubmitSub("Chronicle"))

			got, ok := e.lastArm()
			if !ok {
				t.Fatal("expected an announced MAPLELIFE_ERROR")
			}
			if got != tc.wantArm {
				t.Errorf("arm = %#x, want %#x", got, tc.wantArm)
			}
			if len(e.mapleLifeCalls) != 0 {
				t.Errorf("createMapleLifeFunc calls = %d, want 0", len(e.mapleLifeCalls))
			}
			assertNoDestroySaga(t)
		})
	}
}

func TestMapleLifeCreateSlotLimitBoundary(t *testing.T) {
	cases := []struct {
		charactersInWorld int
		wantProceed       bool
	}{
		{charactersInWorld: 2, wantProceed: true},
		{charactersInWorld: 3, wantProceed: false},
		{charactersInWorld: 4, wantProceed: false},
	}
	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			e := newMapleLifeCreateEnv(t)
			e.accountSlots = 3
			e.charactersInWorld = tc.charactersInWorld

			e.dispatch(mapleLifeSubmitSub("Chronicle"))

			if tc.wantProceed {
				if len(e.mapleLifeCalls) != 1 {
					t.Fatalf("createMapleLifeFunc calls = %d, want 1 (chars=%d, slots=%d)", len(e.mapleLifeCalls), tc.charactersInWorld, e.accountSlots)
				}
			} else {
				if len(e.mapleLifeCalls) != 0 {
					t.Fatalf("createMapleLifeFunc calls = %d, want 0 (chars=%d, slots=%d)", len(e.mapleLifeCalls), tc.charactersInWorld, e.accountSlots)
				}
				got, ok := e.lastArm()
				if !ok || got != mapleLifeCreateByteUnknown {
					t.Errorf("arm = %#x, ok=%v, want the slot-limit arm %#x", got, ok, mapleLifeCreateByteUnknown)
				}
			}
		})
	}
}

// TestMapleLifeCreateBTypeBelowCapIncrementsSlots covers gate 3's B-Type arm
// (bug-b-type-must-add-a-slot.md): B-Type is gated on slots, not on
// character count -- charactersInWorld is left at its zero-value default, so
// if the handler mistakenly ran the A-Type len(chars)>=slots check instead,
// this would still pass and hide the defect. The cap check and increment
// both use the SLOT count, never the character count.
func TestMapleLifeCreateBTypeBelowCapIncrementsSlots(t *testing.T) {
	e := newMapleLifeCreateEnv(t)
	e.slotTemplateId = uint32(item.MapleLifeBTypeId)
	e.accountSlots = character.MaxCharacterSlotsPerWorld - 1
	e.charactersInWorld = 999 // must be ignored: B-Type never reads character count

	e.dispatchItem(item.MapleLifeBTypeId, mapleLifeSubmitSub("Chronicle"))

	if len(e.mapleLifeCalls) != 1 {
		t.Fatalf("createMapleLifeFunc calls = %d, want 1", len(e.mapleLifeCalls))
	}
	if len(e.incrementSlotsCalls) != 1 {
		t.Fatalf("incrementAccountSlotsFunc calls = %d, want 1", len(e.incrementSlotsCalls))
	}
	c := e.incrementSlotsCalls[0]
	if c.accountId != e.s.AccountId() {
		t.Errorf("incrementAccountSlotsFunc accountId = %d, want session's %d", c.accountId, e.s.AccountId())
	}
	if c.worldId != e.s.WorldId() {
		t.Errorf("incrementAccountSlotsFunc worldId = %d, want session's %d", c.worldId, e.s.WorldId())
	}
}

// TestMapleLifeCreateBTypeAtCapIsRejected covers gate 3's B-Type cap arm: at
// the 12-per-world cap, B-Type is rejected and never reaches the factory or
// the increment call, even though charactersInWorld is left well below any
// character-count limit -- proving the cap check reads SLOTS, not the
// character count.
func TestMapleLifeCreateBTypeAtCapIsRejected(t *testing.T) {
	e := newMapleLifeCreateEnv(t)
	e.slotTemplateId = uint32(item.MapleLifeBTypeId)
	e.accountSlots = character.MaxCharacterSlotsPerWorld
	e.charactersInWorld = 0

	e.dispatchItem(item.MapleLifeBTypeId, mapleLifeSubmitSub("Chronicle"))

	if len(e.mapleLifeCalls) != 0 {
		t.Fatalf("createMapleLifeFunc calls = %d, want 0", len(e.mapleLifeCalls))
	}
	if len(e.incrementSlotsCalls) != 0 {
		t.Fatalf("incrementAccountSlotsFunc calls = %d, want 0", len(e.incrementSlotsCalls))
	}
	got, ok := e.lastArm()
	if !ok || got != mapleLifeCreateByteUnknown {
		t.Errorf("arm = %#x, ok=%v, want the slot-limit arm %#x", got, ok, mapleLifeCreateByteUnknown)
	}
}

// TestMapleLifeCreateBTypeFactoryFailureDoesNotIncrement covers the F4
// ordering decision: the increment happens only AFTER a successful factory
// call, so a factory failure must leave the slot count unincremented -- no
// compensating rollback is needed because nothing was written yet.
func TestMapleLifeCreateBTypeFactoryFailureDoesNotIncrement(t *testing.T) {
	e := newMapleLifeCreateEnv(t)
	e.slotTemplateId = uint32(item.MapleLifeBTypeId)
	e.accountSlots = character.MaxCharacterSlotsPerWorld - 1
	e.mapleLifeTransactionId = ""
	e.mapleLifeErr = errors.New("unknown error")

	e.dispatchItem(item.MapleLifeBTypeId, mapleLifeSubmitSub("Chronicle"))

	if len(e.mapleLifeCalls) != 1 {
		t.Fatalf("createMapleLifeFunc calls = %d, want 1", len(e.mapleLifeCalls))
	}
	if len(e.incrementSlotsCalls) != 0 {
		t.Fatalf("incrementAccountSlotsFunc calls = %d, want 0 after a factory failure", len(e.incrementSlotsCalls))
	}
	got, ok := e.lastArm()
	if !ok || got != mapleLifeCreateByteUnknown {
		t.Errorf("arm = %#x, ok=%v, want %#x", got, ok, mapleLifeCreateByteUnknown)
	}
}

// TestMapleLifeCreateRejectsUnroutedItemId covers gate 3's default arm: any
// itemId other than A-Type/B-Type reaching this handler is a routing defect
// (item.GetClassification(itemId) == ClassificationCharacterCreation is the
// only test that should ever route a packet here, and it currently covers
// exactly these two ids), and must fail closed rather than pick a default.
func TestMapleLifeCreateRejectsUnroutedItemId(t *testing.T) {
	e := newMapleLifeCreateEnv(t)
	unrouted := item.Id(5433000)
	e.slotTemplateId = uint32(unrouted)

	e.dispatchItem(unrouted, mapleLifeSubmitSub("Chronicle"))

	if len(e.mapleLifeCalls) != 0 {
		t.Fatalf("createMapleLifeFunc calls = %d, want 0", len(e.mapleLifeCalls))
	}
	if len(e.incrementSlotsCalls) != 0 {
		t.Fatalf("incrementAccountSlotsFunc calls = %d, want 0", len(e.incrementSlotsCalls))
	}
	got, ok := e.lastArm()
	if !ok || got != mapleLifeCreateByteUnknown {
		t.Errorf("arm = %#x, ok=%v, want %#x", got, ok, mapleLifeCreateByteUnknown)
	}
}

// FR-4.5, design C2: the factory's seed path performs no duplicate check of
// its own, so this re-check is the ONLY duplicate gate at submit time -- it
// must run even though Task 12's probe already answered once for this name.
func TestMapleLifeCreateReChecksName(t *testing.T) {
	e := newMapleLifeCreateEnv(t)
	e.dispatch(mapleLifeSubmitSub("Chronicle"))

	if len(e.nameCalls) != 1 {
		t.Fatalf("name-validity calls = %d, want 1", len(e.nameCalls))
	}
	c := e.nameCalls[0]
	if c.name != "Chronicle" {
		t.Errorf("checked name = %q, want %q", c.name, "Chronicle")
	}
	if c.worldId != e.s.WorldId() {
		t.Errorf("worldId = %d, want session's %d", c.worldId, e.s.WorldId())
	}
	if c.scope != character2.NameScopeWorld {
		t.Errorf("scope = %q, want %q", c.scope, character2.NameScopeWorld)
	}
}

// FR-4.2. NOTE: ItemUseMapleLife (Task 6) carries no accountId/worldId field
// on the wire at all (derivation.md §2's field list: sName, al[0..3],
// nGender, nCurrentClass, nSP, update_time) -- unlike the brief's original
// description of this test, there is no packet-carried value to substitute
// or to assert a mismatch-log against. See task-13-report.md. This test is
// therefore narrowed to what IS satisfiable: createMapleLifeFunc must receive
// the SESSION's account/world.
func TestMapleLifeCreateUsesSessionAccountAndWorld(t *testing.T) {
	e := newMapleLifeCreateEnv(t)
	e.dispatch(mapleLifeSubmitSub("Chronicle"))

	if len(e.mapleLifeCalls) != 1 {
		t.Fatalf("createMapleLifeFunc calls = %d, want 1", len(e.mapleLifeCalls))
	}
	c := e.mapleLifeCalls[0]
	if c.accountId != e.s.AccountId() {
		t.Errorf("accountId = %d, want session's %d", c.accountId, e.s.AccountId())
	}
	if c.worldId != e.s.WorldId() {
		t.Errorf("worldId = %d, want session's %d", c.worldId, e.s.WorldId())
	}
}

func TestMapleLifeCreateMapsFactoryOutcomes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		e := newMapleLifeCreateEnv(t)
		e.mapleLifeTransactionId = "tx-1"
		e.mapleLifeErr = nil

		e.dispatch(mapleLifeSubmitSub("Chronicle"))

		if _, ok := e.lastArm(); ok {
			t.Error("expected nothing written to the client on a successful seed -- the outcome awaits the seed event")
		}
		entry, ok := maplelife.GetRegistry().Get(e.tenant(), mapleLifeCreateTestAccountId)
		if !ok {
			t.Fatal("expected a PhaseSubmitted registry entry")
		}
		if entry.Phase != maplelife.PhaseSubmitted {
			t.Errorf("Phase = %q, want %q", entry.Phase, maplelife.PhaseSubmitted)
		}
		if entry.TransactionId != "tx-1" {
			t.Errorf("TransactionId = %q, want %q", entry.TransactionId, "tx-1")
		}
	})

	t.Run("400 invalid look or name", func(t *testing.T) {
		e := newMapleLifeCreateEnv(t)
		e.mapleLifeTransactionId = ""
		e.mapleLifeErr = requests.ErrBadRequest

		e.dispatch(mapleLifeSubmitSub("Chronicle"))

		got, ok := e.lastArm()
		if !ok || got != mapleLifeCreateByteUnknown {
			t.Errorf("arm = %#x, ok=%v, want %#x (only three MAPLELIFE_ERROR arms exist)", got, ok, mapleLifeCreateByteUnknown)
		}
		if _, ok := maplelife.GetRegistry().Get(e.tenant(), mapleLifeCreateTestAccountId); ok {
			t.Error("expected no registry entry after a rejected seed")
		}
	})

	t.Run("500 server error", func(t *testing.T) {
		e := newMapleLifeCreateEnv(t)
		e.mapleLifeTransactionId = ""
		e.mapleLifeErr = errors.New("unknown error")

		e.dispatch(mapleLifeSubmitSub("Chronicle"))

		got, ok := e.lastArm()
		if !ok || got != mapleLifeCreateByteUnknown {
			t.Errorf("arm = %#x, ok=%v, want %#x", got, ok, mapleLifeCreateByteUnknown)
		}
		if _, ok := maplelife.GetRegistry().Get(e.tenant(), mapleLifeCreateTestAccountId); ok {
			t.Error("expected no registry entry after a rejected seed")
		}
	})

	t.Run("transport error", func(t *testing.T) {
		e := newMapleLifeCreateEnv(t)
		e.mapleLifeTransactionId = ""
		e.mapleLifeErr = errors.New("dial tcp: connection refused")

		e.dispatch(mapleLifeSubmitSub("Chronicle"))

		got, ok := e.lastArm()
		if !ok || got != mapleLifeCreateByteUnknown {
			t.Errorf("arm = %#x, ok=%v, want %#x", got, ok, mapleLifeCreateByteUnknown)
		}
		if _, ok := maplelife.GetRegistry().Get(e.tenant(), mapleLifeCreateTestAccountId); ok {
			t.Error("expected no registry entry after a rejected seed")
		}
	})
}

// TestMapleLifeCreateMapsTheWireToTheFactoryRequest is the assertion §11 A8
// says was missing: not just that a seam was called, but that the request the
// channel actually sends carries the al0..al3/gender/currentClass/sp mapping
// (design.md §11 A3). It restores the real createMapleLifeFunc -- the
// harness's own capturing stub would bypass the mapping entirely -- and
// intercepts one level below it, at newFactoryProcessorFunc, so the mapping
// arithmetic runs for real and fakeFactoryProcessor records what it actually
// forwards.
func TestMapleLifeCreateMapsTheWireToTheFactoryRequest(t *testing.T) {
	cases := []struct {
		name             string
		params           mapleLifeSubmitParams
		wantFace         uint32
		wantHair         uint32
		wantHairColor    uint32
		wantSkinColor    byte
		wantGender       byte
		wantClassOrdinal uint32
		wantSP           byte
	}{
		{
			name:             "plain values",
			params:           mapleLifeSubmitParams{name: "Bob", al0: 20000, al1: 30030, al2: 2, al3: 1, gender: 0, currentClass: 0, sp: 5},
			wantFace:         20000,
			wantHair:         30030,
			wantHairColor:    2,
			wantSkinColor:    1,
			wantGender:       0,
			wantClassOrdinal: 0,
			wantSP:           5,
		},
		{
			name:             "hair needs normalising",
			params:           mapleLifeSubmitParams{name: "Bob", al0: 20000, al1: 30034, al2: 2, al3: 1, gender: 0, currentClass: 0, sp: 5},
			wantFace:         20000,
			wantHair:         30030,
			wantHairColor:    2,
			wantSkinColor:    1,
			wantGender:       0,
			wantClassOrdinal: 0,
			wantSP:           5,
		},
		{
			name:             "colour needs normalising",
			params:           mapleLifeSubmitParams{name: "Bob", al0: 20000, al1: 30030, al2: 12, al3: 1, gender: 0, currentClass: 0, sp: 5},
			wantFace:         20000,
			wantHair:         30030,
			wantHairColor:    2,
			wantSkinColor:    1,
			wantGender:       0,
			wantClassOrdinal: 0,
			wantSP:           5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newMapleLifeCreateEnv(t)
			createMapleLifeFunc = originalCreateMapleLifeFunc
			t.Cleanup(func() { createMapleLifeFunc = originalCreateMapleLifeFunc })

			fp := &fakeFactoryProcessor{transactionId: "tx-1"}
			origNewProc := newFactoryProcessorFunc
			newFactoryProcessorFunc = func(_ logrus.FieldLogger, _ context.Context) factory.Processor { return fp }
			t.Cleanup(func() { newFactoryProcessorFunc = origNewProc })

			e.dispatch(mapleLifeSubmitSubWith(tc.params))

			if len(fp.createMapleLifeCalls) != 1 {
				t.Fatalf("CreateMapleLife calls = %d, want 1", len(fp.createMapleLifeCalls))
			}
			c := fp.createMapleLifeCalls[0]
			if c.accountId != e.s.AccountId() {
				t.Errorf("accountId = %d, want session's %d", c.accountId, e.s.AccountId())
			}
			if c.worldId != e.s.WorldId() {
				t.Errorf("worldId = %d, want session's %d", c.worldId, e.s.WorldId())
			}
			if c.name != tc.params.name {
				t.Errorf("name = %q, want %q", c.name, tc.params.name)
			}
			if c.face != tc.wantFace {
				t.Errorf("face = %d, want %d", c.face, tc.wantFace)
			}
			if c.hair != tc.wantHair {
				t.Errorf("hair = %d, want %d", c.hair, tc.wantHair)
			}
			if c.hairColor != tc.wantHairColor {
				t.Errorf("hairColor = %d, want %d", c.hairColor, tc.wantHairColor)
			}
			if c.skinColor != tc.wantSkinColor {
				t.Errorf("skinColor = %d, want %d", c.skinColor, tc.wantSkinColor)
			}
			if c.gender != tc.wantGender {
				t.Errorf("gender = %d, want %d", c.gender, tc.wantGender)
			}
			if c.classOrdinal != tc.wantClassOrdinal {
				t.Errorf("classOrdinal = %d, want %d", c.classOrdinal, tc.wantClassOrdinal)
			}
			if c.sp != tc.wantSP {
				t.Errorf("sp = %d, want %d", c.sp, tc.wantSP)
			}
		})
	}
}

// TestMapleLifeCreateRejectsAnUnconfiguredClassOrdinal covers design.md §11
// A7's gate 5: m_nCurrentClass cycles % 5 in the client's own OnButtonClicked
// (gms_v95 @0x77edc0), so anything outside 0..4 did not come from the dialog
// and is rejected, never clamped.
func TestMapleLifeCreateRejectsAnUnconfiguredClassOrdinal(t *testing.T) {
	cases := []struct {
		name        string
		class       int32
		wantProceed bool
	}{
		{name: "min in range", class: 0, wantProceed: true},
		{name: "max in range", class: 4, wantProceed: true},
		{name: "above range", class: 5, wantProceed: false},
		{name: "negative", class: -1, wantProceed: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newMapleLifeCreateEnv(t)
			e.dispatch(mapleLifeSubmitSubWith(mapleLifeSubmitParams{name: "Chronicle", currentClass: tc.class}))

			if tc.wantProceed {
				if len(e.mapleLifeCalls) != 1 {
					t.Fatalf("createMapleLifeFunc calls = %d, want 1", len(e.mapleLifeCalls))
				}
			} else {
				if len(e.mapleLifeCalls) != 0 {
					t.Fatalf("createMapleLifeFunc calls = %d, want 0", len(e.mapleLifeCalls))
				}
				got, ok := e.lastArm()
				if !ok || got != mapleLifeCreateByteUnknown {
					t.Errorf("arm = %#x, ok=%v, want %#x", got, ok, mapleLifeCreateByteUnknown)
				}
				assertNoDestroySaga(t)
			}
		})
	}
}

// TestMapleLifeCreateRejectsAnOutOfRangeSP covers design.md §11 A7's gate 5:
// m_nSP cycles % 11 in the client's own OnButtonClicked, so anything outside
// 0..10 did not come from the dialog and is rejected, never clamped.
func TestMapleLifeCreateRejectsAnOutOfRangeSP(t *testing.T) {
	cases := []struct {
		name        string
		sp          int32
		wantProceed bool
	}{
		{name: "min in range", sp: 0, wantProceed: true},
		{name: "max in range", sp: 10, wantProceed: true},
		{name: "above range", sp: 11, wantProceed: false},
		{name: "negative", sp: -1, wantProceed: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newMapleLifeCreateEnv(t)
			e.dispatch(mapleLifeSubmitSubWith(mapleLifeSubmitParams{name: "Chronicle", sp: tc.sp}))

			if tc.wantProceed {
				if len(e.mapleLifeCalls) != 1 {
					t.Fatalf("createMapleLifeFunc calls = %d, want 1", len(e.mapleLifeCalls))
				}
			} else {
				if len(e.mapleLifeCalls) != 0 {
					t.Fatalf("createMapleLifeFunc calls = %d, want 0", len(e.mapleLifeCalls))
				}
				got, ok := e.lastArm()
				if !ok || got != mapleLifeCreateByteUnknown {
					t.Errorf("arm = %#x, ok=%v, want %#x", got, ok, mapleLifeCreateByteUnknown)
				}
				assertNoDestroySaga(t)
			}
		})
	}
}

// TestMapleLifeCreateMapsConflictToNameTaken: the factory's own name check
// (design.md §5.2's TOCTOU note -- the window between gate 4's re-check and
// the factory call) surfaces as requests.ErrConflict, which must resolve to
// the same NAME_TAKEN_AT_SUBMIT arm gate 4 itself would have written, not the
// generic UNKNOWN_ERROR every other factory error maps to.
func TestMapleLifeCreateMapsConflictToNameTaken(t *testing.T) {
	e := newMapleLifeCreateEnv(t)
	e.mapleLifeTransactionId = ""
	e.mapleLifeErr = requests.ErrConflict

	e.dispatch(mapleLifeSubmitSub("Chronicle"))

	got, ok := e.lastArm()
	if !ok || got != mapleLifeCreateByteNameTakenSubmit {
		t.Errorf("arm = %#x, ok=%v, want %#x (NAME_TAKEN_AT_SUBMIT)", got, ok, mapleLifeCreateByteNameTakenSubmit)
	}
	if _, ok := maplelife.GetRegistry().Get(e.tenant(), mapleLifeCreateTestAccountId); ok {
		t.Error("expected no registry entry after a rejected create")
	}
}

// FR-5.1: consumption belongs to Task 14's CREATED path alone. Every row of
// every table above exercises a different gate/outcome, so this asserts the
// structural property that backs all of them -- handleMapleLifeCreate's own
// source contains no saga creation of any kind. A seam-based counter cannot
// prove this (there is nothing to intercept if the file never calls
// saga.NewProcessor), so this is a source-scan test, the same technique
// TestMapleLifeArmUsesNoBareCashSlotTypeComparison already uses.
func assertNoDestroySaga(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	src, err := os.ReadFile(filepath.Join(wd, "maple_life_create.go"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	body := string(src)
	if strings.Contains(body, "saga.NewProcessor") || strings.Contains(body, "saga.Saga{") || strings.Contains(body, "DestroyAsset") {
		t.Fatal("maple_life_create.go references the saga package; it must never create a saga (FR-5.1)")
	}
}

func TestMapleLifeCreateNeverConsumesTheItem(t *testing.T) {
	assertNoDestroySaga(t)
}
