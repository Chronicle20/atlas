package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	checkNameTestCharacterId = uint32(7002)
	checkNameTestAccountId   = uint32(5002)
	checkNameTestWorldId     = world.Id(0)
)

// Distinct, non-real resolved bytes, so the assertions prove the handler goes
// through the config-resolved path (DOM-25) rather than a hard-coded byte.
// AVAILABLE is deliberately NOT 0 here for the same reason.
const (
	checkNameByteAvailable = 0x40
	checkNameByteTaken     = 0x41
	checkNameByteUnknown   = 0x42
)

type checkNameHandlerEnv struct {
	t         *testing.T
	ctx       context.Context
	s         session.Model
	l         logrus.FieldLogger
	logs      *bytes.Buffer
	wp        writer.Producer
	announced []struct {
		writer string
		body   []byte
	}

	validity      character.NameValidityResult
	validityErr   error
	validityCalls []struct {
		name    string
		worldId world.Id
		scope   character.NameScope
	}
}

func newCheckNameHandlerEnv(t *testing.T) *checkNameHandlerEnv {
	t.Helper()

	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, discardConn{})
	session.AddSessionToRegistry(ten.Id(), s)
	t.Cleanup(func() { session.ClearRegistryForTenant(ten.Id()) })

	logs := &bytes.Buffer{}
	l := logrus.New()
	l.SetOutput(logs)
	l.SetLevel(logrus.DebugLevel)

	sp := session.NewProcessor(l, ctx)
	sp.SetAccountId(sessionId, checkNameTestAccountId)
	sp.SetCharacterId(sessionId, checkNameTestCharacterId)
	f := field.NewBuilder(checkNameTestWorldId, channel.Id(0), _map.Id(100000000)).Build()
	updated := session.NewProcessor(l, ctx).SetField(sessionId, f)

	env := &checkNameHandlerEnv{t: t, ctx: ctx, s: updated, l: l, logs: logs}
	env.validity = character.NameValidityResult{Valid: true}

	env.wp = func(name string) (swriter.BodyFunc, error) {
		return func(bl logrus.FieldLogger, bctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(bl, bctx)(map[string]interface{}{
					"operations": map[string]interface{}{
						cashcb.CheckNameChangeAvailable:    float64(checkNameByteAvailable),
						cashcb.CheckNameChangeTaken:        float64(checkNameByteTaken),
						cashcb.CheckNameChangeUnknownError: float64(checkNameByteUnknown),
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

	orig := checkNameChangeValidityFunc
	checkNameChangeValidityFunc = func(_ logrus.FieldLogger, _ context.Context, name string, worldId world.Id, scope character.NameScope) (character.NameValidityResult, error) {
		env.validityCalls = append(env.validityCalls, struct {
			name    string
			worldId world.Id
			scope   character.NameScope
		}{name: name, worldId: worldId, scope: scope})
		return env.validity, env.validityErr
	}
	t.Cleanup(func() { checkNameChangeValidityFunc = orig })

	return env
}

func (e *checkNameHandlerEnv) dispatch(name string) {
	e.t.Helper()
	raw := cashsb.NewCheckNameChangeRequest(name).Encode(e.l, e.ctx)(nil)
	req := request.Request(raw)
	r := request.NewRequestReader(&req, 0)
	CashShopCheckNameChangeHandleFunc(e.l, e.ctx, e.wp)(e.s, &r, map[string]interface{}{})
}

// lastResult decodes the CASHSHOP_CHECK_NAME_CHANGE body the handler wrote:
// DecodeStr(sName) + Decode1(nResult).
func (e *checkNameHandlerEnv) lastResult() (string, byte) {
	e.t.Helper()
	if len(e.announced) == 0 {
		e.t.Fatal("no packet was announced")
	}
	a := e.announced[len(e.announced)-1]
	if a.writer != cashcb.CashShopCheckNameChangeWriter {
		e.t.Fatalf("wrote [%s], want [%s]", a.writer, cashcb.CashShopCheckNameChangeWriter)
	}
	if len(a.body) < 3 {
		e.t.Fatalf("body too short: %x", a.body)
	}
	n := int(binary.LittleEndian.Uint16(a.body[0:2]))
	if len(a.body) != 2+n+1 {
		e.t.Fatalf("body length %d, want %d: %x", len(a.body), 2+n+1, a.body)
	}
	return string(a.body[2 : 2+n]), a.body[2+n]
}

// FR-3.5: every cause atlas-character can report reaches the client, and each
// lands on the arm the codec's taxonomy assigns it. Only "duplicate" has a
// distinct client arm (TAKEN); the other three collapse to the client's
// generic error arm, because no GMS build examined has a string for them —
// see cashcb.CheckNameChange's doc comment.
func TestNameChangeCheckReportsEveryUnavailableCause(t *testing.T) {
	cases := []struct {
		name     string
		validity character.NameValidityResult
		want     byte
	}{
		{name: "available", validity: character.NameValidityResult{Valid: true}, want: checkNameByteAvailable},
		{name: "taken in another world", validity: character.NameValidityResult{Reason: character.NameReasonDuplicate}, want: checkNameByteTaken},
		{name: "reserved", validity: character.NameValidityResult{Reason: character.NameReasonReserved}, want: checkNameByteUnknown},
		{name: "too short", validity: character.NameValidityResult{Reason: character.NameReasonLength}, want: checkNameByteUnknown},
		{name: "bad charset", validity: character.NameValidityResult{Reason: character.NameReasonRegex}, want: checkNameByteUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newCheckNameHandlerEnv(t)
			env.validity = tc.validity
			env.dispatch("Uniform")

			gotName, gotCode := env.lastResult()
			if gotName != "Uniform" {
				t.Errorf("echoed name = %q, want %q", gotName, "Uniform")
			}
			if gotCode != tc.want {
				t.Errorf("result byte = %#x, want %#x", gotCode, tc.want)
			}
		})
	}
}

// An unrecognised reason must NOT land on TAKEN. TAKEN renders the specific
// "this name is currently in use" string, and claiming that for a cause we
// have not established would tell the player something untrue.
func TestNameChangeCheckDoesNotClaimTakenForAnUnknownReason(t *testing.T) {
	env := newCheckNameHandlerEnv(t)
	env.validity = character.NameValidityResult{Reason: "something_new_from_atlas_character"}
	env.dispatch("Victor")

	if _, got := env.lastResult(); got != checkNameByteUnknown {
		t.Errorf("result byte = %#x, want %#x (generic error)", got, checkNameByteUnknown)
	}
}

// A failed name-validity call must still answer the client. The rename dialog
// blocks on this result; a silent drop leaves it hung.
func TestNameChangeCheckAnswersWhenTheLookupFails(t *testing.T) {
	env := newCheckNameHandlerEnv(t)
	env.validityErr = errors.New("atlas-character unreachable")
	env.dispatch("Whiskey")

	gotName, gotCode := env.lastResult()
	if gotName != "Whiskey" {
		t.Errorf("echoed name = %q, want %q", gotName, "Whiskey")
	}
	if gotCode != checkNameByteUnknown {
		t.Errorf("result byte = %#x, want %#x", gotCode, checkNameByteUnknown)
	}
}

// FR-3.2: the rename probe checks the WHOLE TENANT, not just the session's
// world. A WORLD-scoped check would let a rename produce a name that already
// exists in another world, which a later world transfer would then collide
// with.
func TestNameChangeCheckUsesTenantScope(t *testing.T) {
	env := newCheckNameHandlerEnv(t)
	env.dispatch("Xray")

	if len(env.validityCalls) != 1 {
		t.Fatalf("name-validity calls = %d, want 1", len(env.validityCalls))
	}
	c := env.validityCalls[0]
	if c.name != "Xray" {
		t.Errorf("checked name = %q, want %q", c.name, "Xray")
	}
	// The session's own world is passed through, though under TENANT scope
	// atlas-character ignores it (name_validity_resource.go). The session
	// package exposes no world setter outside its own package, so this
	// session's world is 0 — the assertion pins the plumbing, not the value.
	if c.worldId != env.s.WorldId() {
		t.Errorf("worldId = %d, want the session's %d", c.worldId, env.s.WorldId())
	}
	if c.scope != character.NameScopeTenant {
		t.Errorf("scope = %q, want %q", c.scope, character.NameScopeTenant)
	}
}

// The reason table is data so this can assert it covers every reason
// atlas-character's CheckNameValidity can return, rather than trusting a
// switch to have listed them.
func TestNameChangeRejectionReasonsCoverAtlasCharacter(t *testing.T) {
	for _, reason := range []string{
		character.NameReasonLength,
		character.NameReasonRegex,
		character.NameReasonDuplicate,
		character.NameReasonReserved,
	} {
		if _, ok := nameChangeRejectionReasons[reason]; !ok {
			t.Errorf("reason %q from atlas-character has no mapping", reason)
		}
	}
}
