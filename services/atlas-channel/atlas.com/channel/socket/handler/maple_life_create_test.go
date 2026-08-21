package handler

import (
	character2 "atlas-channel/character"
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
	mapleLifeCreateTestItemId      = item.Id(5431000) // ClassificationCharacterCreation (543)
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

	accountSlots   int16
	accountSlotErr error

	charactersInWorld    int
	charactersInWorldErr error

	validity    character2.NameValidityResult
	validityErr error
	nameCalls   []struct {
		name    string
		worldId world.Id
		scope   character2.NameScope
	}

	seedTransactionId string
	seedErr           error
	seedCalls         []struct {
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
	env.seedTransactionId = "tx-1"

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
	accountSlotsFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (int16, error) {
		return env.accountSlots, env.accountSlotErr
	}
	t.Cleanup(func() { accountSlotsFunc = origAccountSlots })

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

	origSeed := seedCharacterFunc
	seedCharacterFunc = func(_ logrus.FieldLogger, _ context.Context, accountId uint32, worldId world.Id, sub cashsb.ItemUseMapleLife) (string, error) {
		env.seedCalls = append(env.seedCalls, struct {
			accountId uint32
			worldId   world.Id
			sub       cashsb.ItemUseMapleLife
		}{accountId: accountId, worldId: worldId, sub: sub})
		return env.seedTransactionId, env.seedErr
	}
	t.Cleanup(func() { seedCharacterFunc = origSeed })

	return env
}

func (e *mapleLifeCreateEnv) tenant() tenant.Model {
	return tenant.MustFromContext(e.ctx)
}

// mapleLifeSubmitSub builds a real decoded ItemUseMapleLife the same way the
// classification-543 arm would (character_cash_item_use.go), by round
// tripping through its own wire codec -- littleEndianInt32/encodeAsciiString
// are shared with character_cash_item_use_maple_life_test.go.
func mapleLifeSubmitSub(name string) cashsb.ItemUseMapleLife {
	raw := append([]byte{}, encodeAsciiString(name)...)
	raw = append(raw, littleEndianInt32(0)...) // al0
	raw = append(raw, littleEndianInt32(0)...) // al1
	raw = append(raw, littleEndianInt32(0)...) // al2
	raw = append(raw, littleEndianInt32(0)...) // al3
	raw = append(raw, littleEndianInt32(0)...) // gender
	raw = append(raw, littleEndianInt32(0)...) // currentClass
	raw = append(raw, littleEndianInt32(0)...) // sp
	raw = append(raw, littleEndianInt32(1)...) // updateTime

	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)
	sp := cashsb.NewItemUseMapleLife(false)
	sp.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})
	return *sp
}

func (e *mapleLifeCreateEnv) dispatch(sub cashsb.ItemUseMapleLife) {
	e.t.Helper()
	handleMapleLifeCreate(e.l, e.ctx, e.wp)(e.s, mapleLifeCreateTestItemId, mapleLifeCreateTestSlot, sub)
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
			if len(e.seedCalls) != 0 {
				t.Errorf("seedCharacterFunc calls = %d, want 0", len(e.seedCalls))
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
				if len(e.seedCalls) != 1 {
					t.Fatalf("seedCharacterFunc calls = %d, want 1 (chars=%d, slots=%d)", len(e.seedCalls), tc.charactersInWorld, e.accountSlots)
				}
			} else {
				if len(e.seedCalls) != 0 {
					t.Fatalf("seedCharacterFunc calls = %d, want 0 (chars=%d, slots=%d)", len(e.seedCalls), tc.charactersInWorld, e.accountSlots)
				}
				got, ok := e.lastArm()
				if !ok || got != mapleLifeCreateByteUnknown {
					t.Errorf("arm = %#x, ok=%v, want the slot-limit arm %#x", got, ok, mapleLifeCreateByteUnknown)
				}
			}
		})
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
// therefore narrowed to what IS satisfiable: seedCharacterFunc must receive
// the SESSION's account/world.
func TestMapleLifeCreateUsesSessionAccountAndWorld(t *testing.T) {
	e := newMapleLifeCreateEnv(t)
	e.dispatch(mapleLifeSubmitSub("Chronicle"))

	if len(e.seedCalls) != 1 {
		t.Fatalf("seedCharacterFunc calls = %d, want 1", len(e.seedCalls))
	}
	c := e.seedCalls[0]
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
		e.seedTransactionId = "tx-1"
		e.seedErr = nil

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
		e.seedTransactionId = ""
		e.seedErr = requests.ErrBadRequest

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
		e.seedTransactionId = ""
		e.seedErr = errors.New("unknown error")

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
		e.seedTransactionId = ""
		e.seedErr = errors.New("dial tcp: connection refused")

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
