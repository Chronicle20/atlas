package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	mlcb "github.com/Chronicle20/atlas/libs/atlas-packet/maplelife/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	mapleLifeCheckNameTestCharacterId = uint32(9002)
	mapleLifeCheckNameTestAccountId   = uint32(6002)
	mapleLifeCheckNameTestWorldId     = world.Id(1)
)

// Distinct, non-real resolved bytes, so the assertions prove the handler
// goes through the config-resolved path (DOM-25) rather than a hard-coded
// byte. Available is deliberately NOT 0 here for the same reason
// checkNameByteAvailable is not 0.
const (
	mapleLifeCheckNameByteAvailable = 0x50
	mapleLifeCheckNameByteTaken     = 0x51
	mapleLifeCheckNameByteUnknown   = 0x52
)

type mapleLifeCheckNameEnv struct {
	t         *testing.T
	ctx       context.Context
	s         session.Model
	l         logrus.FieldLogger
	hook      *testlog.Hook
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

func newMapleLifeCheckNameEnv(t *testing.T) *mapleLifeCheckNameEnv {
	t.Helper()

	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	sp := session.NewProcessor(logrus.New(), ctx)
	ch := channel.NewModel(mapleLifeCheckNameTestWorldId, channel.Id(0))
	sp.Create(ch, 0)(sessionId, discardConn{})
	t.Cleanup(func() { session.ClearRegistryForTenant(ten.Id()) })

	sp.SetAccountId(sessionId, mapleLifeCheckNameTestAccountId)
	sp.SetCharacterId(sessionId, mapleLifeCheckNameTestCharacterId)
	f := field.NewBuilder(mapleLifeCheckNameTestWorldId, channel.Id(0), _map.Id(100000000)).Build()
	updated := sp.SetField(sessionId, f)

	l, hook := testlog.NewNullLogger()

	env := &mapleLifeCheckNameEnv{t: t, ctx: ctx, s: updated, l: l, hook: hook}
	env.validity = character.NameValidityResult{Valid: true}

	env.wp = func(name string) (swriter.BodyFunc, error) {
		return func(bl logrus.FieldLogger, bctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(bl, bctx)(map[string]interface{}{
					"operations": map[string]interface{}{
						mlcb.MapleLifeResultAvailable:    float64(mapleLifeCheckNameByteAvailable),
						mlcb.MapleLifeResultTaken:        float64(mapleLifeCheckNameByteTaken),
						mlcb.MapleLifeResultUnknownError: float64(mapleLifeCheckNameByteUnknown),
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

	orig := mapleLifeNameValidityFunc
	mapleLifeNameValidityFunc = func(_ logrus.FieldLogger, _ context.Context, name string, worldId world.Id, scope character.NameScope) (character.NameValidityResult, error) {
		env.validityCalls = append(env.validityCalls, struct {
			name    string
			worldId world.Id
			scope   character.NameScope
		}{name: name, worldId: worldId, scope: scope})
		return env.validity, env.validityErr
	}
	t.Cleanup(func() { mapleLifeNameValidityFunc = orig })

	return env
}

func (e *mapleLifeCheckNameEnv) dispatch(name string) {
	e.t.Helper()
	handleMapleLifeCheckName(e.l, e.ctx, e.wp)(e.s, name)
}

// lastResult decodes the MAPLELIFE_RESULT body the handler wrote:
// EncodeStr(sName) + Decode1(nResult).
func (e *mapleLifeCheckNameEnv) lastResult() (string, byte) {
	e.t.Helper()
	if len(e.announced) == 0 {
		e.t.Fatal("no packet was announced")
	}
	a := e.announced[len(e.announced)-1]
	if a.writer != mlcb.MapleLifeResultWriter {
		e.t.Fatalf("wrote [%s], want [%s]", a.writer, mlcb.MapleLifeResultWriter)
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

// FR-3.2 analogue for creation: the probe checks the SESSION'S WORLD, not the
// tenant, because character creation only cares whether the name collides
// within the world the character will be created in. TENANT is the
// deliberately stricter rename-only scope (name_validity_requests.go:19-27).
func TestMapleLifeCheckNameAsksForWorldScope(t *testing.T) {
	env := newMapleLifeCheckNameEnv(t)
	env.dispatch("Yankee")

	if len(env.validityCalls) != 1 {
		t.Fatalf("name-validity calls = %d, want 1", len(env.validityCalls))
	}
	c := env.validityCalls[0]
	if c.name != "Yankee" {
		t.Errorf("checked name = %q, want %q", c.name, "Yankee")
	}
	if c.worldId != env.s.WorldId() {
		t.Errorf("worldId = %d, want the session's %d", c.worldId, env.s.WorldId())
	}
	if c.scope != character.NameScopeWorld {
		t.Errorf("scope = %q, want %q", c.scope, character.NameScopeWorld)
	}
}

// Every cause atlas-character can report reaches the client through the real
// codec path (mlcb.MapleLifeResultRejectedBody), not a handler-local copy of
// the reason table.
func TestMapleLifeCheckNameMapsReasons(t *testing.T) {
	cases := []struct {
		name       string
		validity   character.NameValidityResult
		validityEr error
		want       byte
		wantLevel  logrus.Level
	}{
		{name: "available", validity: character.NameValidityResult{Valid: true}, want: mapleLifeCheckNameByteAvailable},
		{name: "duplicate", validity: character.NameValidityResult{Reason: character.NameReasonDuplicate}, want: mapleLifeCheckNameByteTaken, wantLevel: logrus.InfoLevel},
		{name: "reserved", validity: character.NameValidityResult{Reason: character.NameReasonReserved}, want: mapleLifeCheckNameByteUnknown, wantLevel: logrus.InfoLevel},
		{name: "length", validity: character.NameValidityResult{Reason: character.NameReasonLength}, want: mapleLifeCheckNameByteUnknown, wantLevel: logrus.InfoLevel},
		{name: "regex", validity: character.NameValidityResult{Reason: character.NameReasonRegex}, want: mapleLifeCheckNameByteUnknown, wantLevel: logrus.InfoLevel},
		{name: "unknown reason", validity: character.NameValidityResult{Reason: "banana"}, want: mapleLifeCheckNameByteUnknown, wantLevel: logrus.ErrorLevel},
		{name: "seam error", validityEr: errors.New("boom"), want: mapleLifeCheckNameByteUnknown, wantLevel: logrus.ErrorLevel},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newMapleLifeCheckNameEnv(t)
			env.validity = tc.validity
			env.validityErr = tc.validityEr
			env.dispatch("Zulu")

			gotName, gotCode := env.lastResult()
			if gotName != "Zulu" {
				t.Errorf("echoed name = %q, want %q", gotName, "Zulu")
			}
			if gotCode != tc.want {
				t.Errorf("result byte = %#x, want %#x", gotCode, tc.want)
			}
			if tc.wantLevel != 0 {
				found := false
				for _, e := range env.hook.AllEntries() {
					if e.Level == tc.wantLevel {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected a %s-level log entry, got: %+v", tc.wantLevel, env.hook.AllEntries())
				}
			}
		})
	}
}

// The reason table this handler relies on (mlcb.MapleLifeResultRejectedBody,
// libs/atlas-packet/maplelife/clientbound/result.go) is asserted exhaustive
// at its own layer by TestMapleLifeResultReasonMapping (result_test.go). That
// map is unexported there, so it cannot be inspected from this package;
// instead this drives handleMapleLifeCheckName once per
// character.NameReason* constant and asserts each produces a REJECTION (a
// non-AVAILABLE arm), which is the property this handler actually depends on.
func TestMapleLifeCheckNameReasonTableIsExhaustive(t *testing.T) {
	for _, reason := range []string{
		character.NameReasonLength,
		character.NameReasonRegex,
		character.NameReasonDuplicate,
		character.NameReasonReserved,
	} {
		t.Run(reason, func(t *testing.T) {
			env := newMapleLifeCheckNameEnv(t)
			env.validity = character.NameValidityResult{Reason: reason}
			env.dispatch("Alpha")

			_, gotCode := env.lastResult()
			if gotCode == mapleLifeCheckNameByteAvailable {
				t.Errorf("reason %q resolved to AVAILABLE, want a rejection arm", reason)
			}
		})
	}
}
