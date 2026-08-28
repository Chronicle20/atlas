package handler

import (
	"atlas-channel/character"
	"atlas-channel/maps/location"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	chat "github.com/Chronicle20/atlas/libs/atlas-packet/chat/serverbound"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	findRequesterId   = uint32(7100)
	findRequesterAcct = uint32(5100)
	findTargetId      = uint32(7200)
	findWorldId       = world.Id(0)
	// The requester sits on channel 2 and the remote target on channel 7.
	// 7 is deliberately neither 0 nor 1: the bug being fixed is a hard-coded 0
	// on the channel arm, and the client adds one for display, so a fixture on
	// 0 or 1 passes against the broken code.
	findRequesterChannel = channel.Id(2)
	findRemoteChannel    = channel.Id(7)
)

type findEnv struct {
	t         *testing.T
	ctx       context.Context
	s         session.Model
	l         logrus.FieldLogger
	logs      *bytes.Buffer
	wp        writer.Producer
	announced [][]byte

	// seam returns
	target    character.Model
	targetErr error
	localSess session.Model
	localErr  error
	loc       location.Model
	locErr    error
	locCalls  int
	tenantId  uuid.UUID
	sessionId uuid.UUID
}

func newFindEnv(t *testing.T) *findEnv {
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
	sp.SetAccountId(sessionId, findRequesterAcct)
	sp.SetCharacterId(sessionId, findRequesterId)
	f := field.NewBuilder(findWorldId, findRequesterChannel, _map.Id(100000000)).Build()
	updated := session.NewProcessor(l, ctx).SetField(sessionId, f)

	env := &findEnv{t: t, ctx: ctx, s: updated, l: l, logs: logs, tenantId: ten.Id(), sessionId: sessionId}

	env.wp = func(name string) (swriter.BodyFunc, error) {
		return func(bl logrus.FieldLogger, bctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(bl, bctx)(nil)
				env.announced = append(env.announced, b)
				return b
			}
		}, nil
	}

	// Defaults: the target resolves, is in the requester's world, is not a GM,
	// has no local session, and has no location row. Each subtest overrides
	// only what its rule needs.
	env.target = character.NewBuilder().
		SetId(findTargetId).
		SetName("Bob").
		SetWorldId(findWorldId).
		SetGm(0).
		SetX(250).
		SetY(-75).
		MustBuild()
	env.localErr = errors.New("no local session")
	env.locErr = location.ErrNotFound

	origName := findCharacterByNameFunc
	findCharacterByNameFunc = func(_ logrus.FieldLogger, _ context.Context, _ string) (character.Model, error) {
		return env.target, env.targetErr
	}
	t.Cleanup(func() { findCharacterByNameFunc = origName })

	origSess := findLocalSessionFunc
	findLocalSessionFunc = func(_ logrus.FieldLogger, _ context.Context, _ channel.Model, _ uint32) (session.Model, error) {
		return env.localSess, env.localErr
	}
	t.Cleanup(func() { findLocalSessionFunc = origSess })

	origLoc := findCharacterLocationFunc
	findCharacterLocationFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (location.Model, error) {
		env.locCalls++
		return env.loc, env.locErr
	}
	t.Cleanup(func() { findCharacterLocationFunc = origLoc })

	return env
}

// dispatch drives the real handler for one arm and returns the announced body.
func (e *findEnv) dispatch(mode chat.WhisperMode, targetName string) []byte {
	e.t.Helper()
	e.announced = nil
	if err := produceFindResultBody(e.l)(e.ctx)(e.wp)(mode, targetName)(e.s); err != nil {
		e.t.Fatalf("produceFindResultBody: %v", err)
	}
	if len(e.announced) != 1 {
		e.t.Fatalf("announced %d packets, want 1", len(e.announced))
	}
	return e.announced[0]
}

// The two find arms. 0x09 is the chat /find; 0x48 is the buddy-window find.
var findArms = []struct {
	name string
	mode chat.WhisperMode
	echo byte
}{
	{"chat arm", chat.WhisperModeFind, 0x09},
	{"buddy window arm", chat.WhisperModeBuddyWindowFind, 0x48},
}

// decodeFindError decodes an error-shape body and returns the echoed name.
func decodeFindError(t *testing.T, ctx context.Context, l logrus.FieldLogger, b []byte) (byte, string) {
	t.Helper()
	req := request.Request(b)
	reader := request.NewRequestReader(&req, 0)
	var m fieldcb.WhisperFindResultError
	m.Decode(l, ctx)(&reader, nil)
	return m.Mode(), m.TargetName()
}

func decodeFindChannel(t *testing.T, ctx context.Context, l logrus.FieldLogger, b []byte) fieldcb.WhisperFindResultChannel {
	t.Helper()
	req := request.Request(b)
	reader := request.NewRequestReader(&req, 0)
	var m fieldcb.WhisperFindResultChannel
	m.Decode(l, ctx)(&reader, nil)
	return m
}

func decodeFindCashShop(t *testing.T, ctx context.Context, l logrus.FieldLogger, b []byte) fieldcb.WhisperFindResultCashShop {
	t.Helper()
	req := request.Request(b)
	reader := request.NewRequestReader(&req, 0)
	var m fieldcb.WhisperFindResultCashShop
	m.Decode(l, ctx)(&reader, nil)
	return m
}

// FR-1: an unresolvable name answers the error shape and echoes the REQUESTED
// name — there is no resolved one.
func TestFind_FR1_UnresolvableName(t *testing.T) {
	for _, arm := range findArms {
		t.Run(arm.name, func(t *testing.T) {
			env := newFindEnv(t)
			env.targetErr = errors.New("no such character")

			mode, name := decodeFindError(t, env.ctx, env.l, env.dispatch(arm.mode, "Ghost"))
			if mode != arm.echo {
				t.Errorf("mode = %#x, want %#x", mode, arm.echo)
			}
			if name != "Ghost" {
				t.Errorf("echoed name = %q, want the requested name %q", name, "Ghost")
			}
			if !bytes.Contains(env.logs.Bytes(), []byte("unresolved")) {
				t.Errorf("no branch=unresolved log line in %s", env.logs.String())
			}
		})
	}
}

// FR-2: a cross-world target is wire-identical to FR-1 by design (PRD §8) but
// must be a DISTINCT branch — otherwise it is FR-1 passing by accident.
func TestFind_FR2_CrossWorld(t *testing.T) {
	for _, arm := range findArms {
		t.Run(arm.name, func(t *testing.T) {
			env := newFindEnv(t)
			env.target = character.NewBuilder().
				SetId(findTargetId).SetName("Bob").SetWorldId(world.Id(1)).SetGm(0).MustBuild()

			mode, name := decodeFindError(t, env.ctx, env.l, env.dispatch(arm.mode, "Bob"))
			if mode != arm.echo {
				t.Errorf("mode = %#x, want %#x", mode, arm.echo)
			}
			if name != "Bob" {
				t.Errorf("echoed name = %q, want Bob", name)
			}
			if !bytes.Contains(env.logs.Bytes(), []byte("cross-world")) {
				t.Errorf("no branch=cross-world log line in %s", env.logs.String())
			}
			if env.locCalls != 0 {
				t.Errorf("location looked up %d times for a cross-world target, want 0", env.locCalls)
			}
		})
	}
}

// FR-3: GM visibility is a boolean predicate on level > 0, not a tier
// comparison. The gm=2 case is the whole point — it passes against the old
// `gm == 1` predicate, which classified a level-2 GM as an ordinary player.
func TestFind_FR3_GmConcealment(t *testing.T) {
	cases := []struct {
		name          string
		targetGm      int
		requesterGm   bool
		wantConcealed bool
	}{
		{"gm 1 target, ordinary requester", 1, false, true},
		{"gm 2 target, ordinary requester", 2, false, true},
		{"gm 1 target, gm requester", 1, true, false},
		{"gm 2 target, gm requester", 2, true, false},
		{"ordinary target, ordinary requester", 0, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := newFindEnv(t)
			env.target = character.NewBuilder().
				SetId(findTargetId).SetName("Bob").SetWorldId(findWorldId).SetGm(c.targetGm).MustBuild()
			if c.requesterGm {
				env.s = session.NewProcessor(env.l, env.ctx).SetGm(env.sessionId, true)
			}
			env.loc = locationFixture(characterconst.PresenceStateInField, findRemoteChannel)
			env.locErr = nil

			b := env.dispatch(chat.WhisperModeFind, "Bob")
			if c.wantConcealed {
				if _, name := decodeFindError(t, env.ctx, env.l, b); name != "Bob" {
					t.Errorf("echoed name = %q, want Bob", name)
				}
				if !bytes.Contains(env.logs.Bytes(), []byte("gm-concealed")) {
					t.Errorf("no branch=gm-concealed log line in %s", env.logs.String())
				}
				return
			}
			if got := decodeFindChannel(t, env.ctx, env.l, b); got.ChannelId() != uint32(findRemoteChannel) {
				t.Errorf("channel = %d, want %d", got.ChannelId(), findRemoteChannel)
			}
		})
	}
}

// FR-4a: a live local session in a cash scene answers the cash-shop shape
// without consulting the location row at all.
func TestFind_FR4a_LocalCashScene(t *testing.T) {
	scenes := []struct {
		name  string
		scene byte
	}{
		{"cash shop", session.CashSceneCashShop},
		// The ITC renders inside the cash-shop CStage and emits the identical
		// CHARACTER_ENTER, so MTS folds into the same shape (FR-11).
		{"mts", session.CashSceneMts},
	}
	for _, sc := range scenes {
		for _, arm := range findArms {
			t.Run(sc.name+" "+arm.name, func(t *testing.T) {
				env := newFindEnv(t)
				targetSessionId := uuid.New()
				ts := session.NewSession(targetSessionId, mustTenant(t, "GMS", 83, 1), 0, discardConn{})
				session.AddSessionToRegistry(env.tenantId, ts)
				env.localSess = session.NewProcessor(env.l, env.ctx).SetCashScene(targetSessionId, sc.scene)
				env.localErr = nil

				got := decodeFindCashShop(t, env.ctx, env.l, env.dispatch(arm.mode, "Bob"))
				if got.Mode() != arm.echo {
					t.Errorf("mode = %#x, want %#x", got.Mode(), arm.echo)
				}
				if got.TargetName() != "Bob" {
					t.Errorf("name = %q, want Bob", got.TargetName())
				}
				if env.locCalls != 0 {
					t.Errorf("location looked up %d times for a local cash-scene target, want 0", env.locCalls)
				}
			})
		}
	}
}

// FR-5: a live local session NOT in a cash scene answers the map shape. The
// 0x09 arm carries x/y; the 0x48 arm does not.
func TestFind_FR5_LocalOnMap(t *testing.T) {
	for _, arm := range findArms {
		t.Run(arm.name, func(t *testing.T) {
			env := newFindEnv(t)
			targetSessionId := uuid.New()
			ts := session.NewSession(targetSessionId, mustTenant(t, "GMS", 83, 1), 0, discardConn{})
			session.AddSessionToRegistry(env.tenantId, ts)
			env.localSess = session.NewProcessor(env.l, env.ctx).SetCashScene(targetSessionId, session.CashSceneNone)
			env.localErr = nil
			env.loc = locationFixture(characterconst.PresenceStateInField, findRequesterChannel)
			env.locErr = nil

			var m fieldcb.WhisperFindResultMap
			b := env.dispatch(arm.mode, "Bob")
			req := request.Request(b)
			reader := request.NewRequestReader(&req, 0)
			m.Decode(env.l, env.ctx)(&reader, nil)
			if m.Mode() != arm.echo {
				t.Errorf("mode = %#x, want %#x", m.Mode(), arm.echo)
			}
			if m.MapId() != 100000000 {
				t.Errorf("mapId = %d, want 100000000", m.MapId())
			}

			// Assert the x/y presence by wire length rather than by decoding,
			// because WhisperFindResultMap.Decode reads x/y only when the
			// receiving struct already has includeXY set.
			withoutXY := len(b)
			if arm.echo == 0x09 {
				if withoutXY != mapBodyLen("Bob")+8 {
					t.Errorf("0x09 body is %d bytes; want map body + 8 for x/y", withoutXY)
				}
			} else if withoutXY != mapBodyLen("Bob") {
				t.Errorf("0x48 body is %d bytes; want map body with NO x/y", withoutXY)
			}
		})
	}
}

// FR-4b: no local session, but the location row says the target is in the
// cash shop.
func TestFind_FR4b_RemoteCashShop(t *testing.T) {
	for _, arm := range findArms {
		t.Run(arm.name, func(t *testing.T) {
			env := newFindEnv(t)
			env.loc = locationFixture(characterconst.PresenceStateInCashShop, findRemoteChannel)
			env.locErr = nil

			got := decodeFindCashShop(t, env.ctx, env.l, env.dispatch(arm.mode, "Bob"))
			if got.Mode() != arm.echo || got.TargetName() != "Bob" {
				t.Errorf("mode=%#x name=%q, want %#x Bob", got.Mode(), got.TargetName(), arm.echo)
			}
			if !bytes.Contains(env.logs.Bytes(), []byte("cash-shop-remote")) {
				t.Errorf("no branch=cash-shop-remote log line in %s", env.logs.String())
			}
		})
	}
}

// FR-6: the bug this task exists to fix. The channel arm must carry the
// target's REAL channel, not a hard-coded 0.
func TestFind_FR6_RemoteChannel(t *testing.T) {
	for _, arm := range findArms {
		t.Run(arm.name, func(t *testing.T) {
			env := newFindEnv(t)
			env.loc = locationFixture(characterconst.PresenceStateInField, findRemoteChannel)
			env.locErr = nil

			got := decodeFindChannel(t, env.ctx, env.l, env.dispatch(arm.mode, "Bob"))
			if got.Mode() != arm.echo {
				t.Errorf("mode = %#x, want %#x", got.Mode(), arm.echo)
			}
			if got.ChannelId() != uint32(findRemoteChannel) {
				t.Errorf("channelId = %d, want %d (a hard-coded 0 must not pass)", got.ChannelId(), findRemoteChannel)
			}
		})
	}
}

// FR-7: the three not-findable branches. All three answer the same wire shape
// and differ only in the branch name and log level.
func TestFind_FR7_NotFindable(t *testing.T) {
	cases := []struct {
		name       string
		setup      func(*findEnv)
		wantBranch string
	}{
		{
			name: "offline",
			setup: func(e *findEnv) {
				e.loc = locationFixture(characterconst.PresenceStateOffline, findRemoteChannel)
				e.locErr = nil
			},
			wantBranch: "offline",
		},
		{
			name:       "never logged in",
			setup:      func(e *findEnv) { e.locErr = location.ErrNotFound },
			wantBranch: "never-logged-in",
		},
		{
			name:       "lookup failed",
			setup:      func(e *findEnv) { e.locErr = errors.New("500 from atlas-maps") },
			wantBranch: "lookup-failed",
		},
	}
	for _, c := range cases {
		for _, arm := range findArms {
			t.Run(c.name+" "+arm.name, func(t *testing.T) {
				env := newFindEnv(t)
				c.setup(env)

				mode, name := decodeFindError(t, env.ctx, env.l, env.dispatch(arm.mode, "Bob"))
				if mode != arm.echo {
					t.Errorf("mode = %#x, want %#x", mode, arm.echo)
				}
				if name != "Bob" {
					t.Errorf("echoed name = %q, want Bob", name)
				}
				if !bytes.Contains(env.logs.Bytes(), []byte(c.wantBranch)) {
					t.Errorf("no branch=%s log line in %s", c.wantBranch, env.logs.String())
				}
				if c.wantBranch == "lookup-failed" && !bytes.Contains(env.logs.Bytes(), []byte("level=error")) {
					t.Errorf("infrastructure failure was not logged at error level: %s", env.logs.String())
				}
			})
		}
	}
}

// The two arms differ only in the echoed mode byte for every outcome except
// the map shape, where 0x09 additionally carries x/y.
func TestFind_ArmSymmetry(t *testing.T) {
	setups := map[string]func(*findEnv){
		"cash shop": func(e *findEnv) {
			e.loc = locationFixture(characterconst.PresenceStateInCashShop, findRemoteChannel)
			e.locErr = nil
		},
		"channel": func(e *findEnv) {
			e.loc = locationFixture(characterconst.PresenceStateInField, findRemoteChannel)
			e.locErr = nil
		},
		"error": func(e *findEnv) { e.locErr = location.ErrNotFound },
	}
	for name, setup := range setups {
		t.Run(name, func(t *testing.T) {
			envA := newFindEnv(t)
			setup(envA)
			a := envA.dispatch(chat.WhisperModeFind, "Bob")

			envB := newFindEnv(t)
			setup(envB)
			b := envB.dispatch(chat.WhisperModeBuddyWindowFind, "Bob")

			if len(a) != len(b) {
				t.Fatalf("arm bodies differ in length: 0x09 %d, 0x48 %d", len(a), len(b))
			}
			if a[0] != 0x09 || b[0] != 0x48 {
				t.Fatalf("mode bytes = %#x / %#x, want 0x09 / 0x48", a[0], b[0])
			}
			if !bytes.Equal(a[1:], b[1:]) {
				t.Errorf("arm bodies differ past the mode byte:\n 0x09 %v\n 0x48 %v", a[1:], b[1:])
			}
		})
	}
}

// decodeWhisperSendResult decodes a WhisperSendResult body and returns the
// echoed name and success flag.
func decodeWhisperSendResult(t *testing.T, ctx context.Context, l logrus.FieldLogger, b []byte) (string, bool) {
	t.Helper()
	req := request.Request(b)
	reader := request.NewRequestReader(&req, 0)
	var m fieldcb.WhisperSendResult
	m.Decode(l, ctx)(&reader, nil)
	return m.TargetName(), m.Success()
}

// dispatchWhisperChat drives produceWhisperChatResult directly, mirroring
// findEnv.dispatch, and returns the announced body (nil if nothing was
// announced, which is the "produced, success comes from the Kafka round
// trip" case).
func (e *findEnv) dispatchWhisperChat(msg, targetName string) []byte {
	e.t.Helper()
	e.announced = nil
	if err := produceWhisperChatResult(e.l)(e.ctx)(e.wp)(msg, targetName)(e.s); err != nil {
		e.t.Fatalf("produceWhisperChatResult: %v", err)
	}
	if len(e.announced) == 0 {
		return nil
	}
	if len(e.announced) != 1 {
		e.t.Fatalf("announced %d packets, want 0 or 1", len(e.announced))
	}
	return e.announced[0]
}

// TestWhisperChat_Decision covers every row of the WhisperModeChat decision
// table: whether WhisperSendResult(false) is announced without producing the
// chat command, or the chat command is produced (success comes from the
// Kafka round trip, not from here).
func TestWhisperChat_Decision(t *testing.T) {
	cases := []struct {
		name           string
		setup          func(*findEnv)
		wantBranch     string
		wantAnnounced  bool // announced WhisperSendResult(false)
		wantProduced   bool
		wantErrorLevel bool
	}{
		{
			name:          "unresolvable name",
			setup:         func(e *findEnv) { e.targetErr = errors.New("no such character") },
			wantBranch:    "unresolved",
			wantAnnounced: true,
			wantProduced:  false,
		},
		{
			name:          "never logged in",
			setup:         func(e *findEnv) { e.locErr = location.ErrNotFound },
			wantBranch:    "never-logged-in",
			wantAnnounced: true,
			wantProduced:  false,
		},
		{
			name: "offline",
			setup: func(e *findEnv) {
				e.loc = locationFixture(characterconst.PresenceStateOffline, findRemoteChannel)
				e.locErr = nil
			},
			wantBranch:    "offline",
			wantAnnounced: true,
			wantProduced:  false,
		},
		{
			name: "in cash shop",
			setup: func(e *findEnv) {
				e.loc = locationFixture(characterconst.PresenceStateInCashShop, findRemoteChannel)
				e.locErr = nil
			},
			wantBranch:    "cash-shop",
			wantAnnounced: true,
			wantProduced:  false,
		},
		{
			name: "in field",
			setup: func(e *findEnv) {
				e.loc = locationFixture(characterconst.PresenceStateInField, findRemoteChannel)
				e.locErr = nil
			},
			wantBranch:    "in-field",
			wantAnnounced: false,
			wantProduced:  true,
		},
		{
			name:           "infrastructure error fails open",
			setup:          func(e *findEnv) { e.locErr = errors.New("500 from atlas-maps") },
			wantBranch:     "lookup-failed",
			wantAnnounced:  false,
			wantProduced:   true,
			wantErrorLevel: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := newFindEnv(t)
			c.setup(env)

			var produced bool
			origProduce := produceWhisperChatFunc
			produceWhisperChatFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _ string, _ string) error {
				produced = true
				return nil
			}
			t.Cleanup(func() { produceWhisperChatFunc = origProduce })

			b := env.dispatchWhisperChat("hi", "Bob")

			if produced != c.wantProduced {
				t.Errorf("produced = %v, want %v", produced, c.wantProduced)
			}
			if c.wantAnnounced {
				if b == nil {
					t.Fatalf("no packet announced, want WhisperSendResult(false)")
				}
				name, success := decodeWhisperSendResult(t, env.ctx, env.l, b)
				if name != "Bob" {
					t.Errorf("echoed name = %q, want Bob", name)
				}
				if success {
					t.Errorf("success = true, want false")
				}
			} else if b != nil {
				t.Errorf("announced a packet, want none: %v", b)
			}
			if !bytes.Contains(env.logs.Bytes(), []byte(c.wantBranch)) {
				t.Errorf("no branch=%s log line in %s", c.wantBranch, env.logs.String())
			}
			if c.wantErrorLevel && !bytes.Contains(env.logs.Bytes(), []byte("level=error")) {
				t.Errorf("infrastructure failure was not logged at error level: %s", env.logs.String())
			}
		})
	}
}

// TestWhisperChat_ProduceFailure asserts the existing behaviour is preserved:
// a deliverable target whose chat command production itself fails still
// answers WhisperSendResult(false).
func TestWhisperChat_ProduceFailure(t *testing.T) {
	env := newFindEnv(t)
	env.loc = locationFixture(characterconst.PresenceStateInField, findRemoteChannel)
	env.locErr = nil

	origProduce := produceWhisperChatFunc
	produceWhisperChatFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _ string, _ string) error {
		return errors.New("kafka unavailable")
	}
	t.Cleanup(func() { produceWhisperChatFunc = origProduce })

	b := env.dispatchWhisperChat("hi", "Bob")
	if b == nil {
		t.Fatalf("no packet announced, want WhisperSendResult(false)")
	}
	name, success := decodeWhisperSendResult(t, env.ctx, env.l, b)
	if name != "Bob" || success {
		t.Errorf("name=%q success=%v, want Bob/false", name, success)
	}
}

// locationFixture builds a location.Model in the given state on the given
// channel, at map 100000000.
func locationFixture(state characterconst.PresenceState, ch channel.Id) location.Model {
	return location.NewModelForTest(findTargetId, findWorldId, ch, _map.Id(100000000), uuid.Nil, state)
}

// mapBodyLen returns the byte length of a map-shape body without x/y:
// mode(1) + ascii string(2 + len) + findMode(1) + mapId(4).
func mapBodyLen(name string) int {
	return 1 + 2 + len(name) + 1 + 4
}
