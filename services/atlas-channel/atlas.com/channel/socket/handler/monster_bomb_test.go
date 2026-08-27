package handler

import (
	"atlas-channel/character"
	"atlas-channel/monster"
	"atlas-channel/session"
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const monsterBombMonsterId = uint32(7001)

// monsterBombPacket is the wire-encoded MONSTER_BOMB payload for mob 7001:
// a single Encode4, so 7001 is the four little-endian bytes below.
var monsterBombPacket = []byte{0x59, 0x1B, 0x00, 0x00}

// monsterBombEnv wires a session in a given field plus swapped seams for one
// MonsterBombHandleFunc dispatch.
type monsterBombEnv struct {
	t    *testing.T
	ctx  context.Context
	s    session.Model
	l    logrus.FieldLogger
	logs *bytes.Buffer

	character character.Model
	charErr   error

	selfDestructCalls int
	lastField         field.Model
	lastMonsterId     uint32
	lastCharacterId   uint32
	selfDestructErr   error
}

func newMonsterBombEnv(t *testing.T, f field.Model) *monsterBombEnv {
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
	sp.SetCharacterId(sessionId, monsterBombCharacterId)
	updated := session.NewProcessor(l, ctx).SetField(sessionId, f)

	env := &monsterBombEnv{t: t, ctx: ctx, s: updated, l: l, logs: logs}

	env.character = character.NewBuilder().
		SetId(monsterBombCharacterId).
		SetHp(500).
		MustBuild()

	origChar := monsterBombCharacterFunc
	monsterBombCharacterFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (character.Model, error) {
		return env.character, env.charErr
	}
	t.Cleanup(func() { monsterBombCharacterFunc = origChar })

	origSelfDestruct := monsterBombSelfDestructFunc
	monsterBombSelfDestructFunc = func(_ logrus.FieldLogger, _ context.Context, f field.Model, monsterId uint32, characterId uint32) error {
		env.selfDestructCalls++
		env.lastField = f
		env.lastMonsterId = monsterId
		env.lastCharacterId = characterId
		return env.selfDestructErr
	}
	t.Cleanup(func() { monsterBombSelfDestructFunc = origSelfDestruct })

	return env
}

// dispatch drives the real handler with the standard mob-7001 packet.
func (e *monsterBombEnv) dispatch() {
	e.t.Helper()
	req := request.Request(monsterBombPacket)
	reader := request.NewRequestReader(&req, 0)
	MonsterBombHandleFunc(e.l, e.ctx, nil)(e.s, &reader, nil)
}

const monsterBombCharacterId = uint32(4242)

// monsterBombField matches session.ProcessorImpl.SetField, which sets only
// the map and instance on the session's field — worldId/channelId stay at the
// session's zero-value default — so worldId/channelId here must be 0 too, or
// s.Field() would never equal a field built with a non-zero channel.
func monsterBombField() field.Model {
	return field.NewBuilder(world.Id(0), 0, _map.Id(100000000)).Build()
}

// TestMonsterBombDetonates covers the happy path: a live character, in the
// same field as a monster the mirror knows about, produces exactly one
// self-destruct request carrying the reporter's field, the monster's id, and
// the reporter's character id.
func TestMonsterBombDetonates(t *testing.T) {
	f := monsterBombField()
	env := newMonsterBombEnv(t, f)

	ten := tenant.MustFromContext(env.ctx)
	monster.GetLiveMirror().Put(ten, monsterBombMonsterId, monster.LiveEntry{Field: f, MonsterId: 8500003})

	env.dispatch()

	if env.selfDestructCalls != 1 {
		t.Fatalf("selfDestructCalls = %d, want 1", env.selfDestructCalls)
	}
	if !env.lastField.Equals(f) {
		t.Errorf("field = %v, want %v", env.lastField, f)
	}
	if env.lastMonsterId != monsterBombMonsterId {
		t.Errorf("monsterId = %d, want %d", env.lastMonsterId, monsterBombMonsterId)
	}
	if env.lastCharacterId != monsterBombCharacterId {
		t.Errorf("characterId = %d, want %d", env.lastCharacterId, monsterBombCharacterId)
	}
}

// TestMonsterBombRejects covers every guard the channel enforces before
// forwarding a detonation request; in each case the seam must not be called.
func TestMonsterBombRejects(t *testing.T) {
	cases := []struct {
		name       string
		setup      func(t *testing.T, env *monsterBombEnv, f field.Model)
		wantSubstr string
	}{
		{
			name: "character lookup fails",
			setup: func(t *testing.T, env *monsterBombEnv, f field.Model) {
				env.charErr = errors.New("500 from atlas-character")
			},
			wantSubstr: "unable to resolve",
		},
		{
			name: "dead character",
			setup: func(t *testing.T, env *monsterBombEnv, f field.Model) {
				env.character = character.NewBuilder().SetId(monsterBombCharacterId).SetHp(0).MustBuild()
			},
			wantSubstr: "while dead",
		},
		{
			name: "mirror miss",
			setup: func(t *testing.T, env *monsterBombEnv, f field.Model) {
				// no Put call: the mirror never learns about 7001
			},
			wantSubstr: "not in the live mirror",
		},
		{
			name: "mob in another field",
			setup: func(t *testing.T, env *monsterBombEnv, f field.Model) {
				other := field.NewBuilder(world.Id(0), 0, _map.Id(200000000)).Build()
				ten := tenant.MustFromContext(env.ctx)
				monster.GetLiveMirror().Put(ten, monsterBombMonsterId, monster.LiveEntry{Field: other, MonsterId: 8500003})
			},
			wantSubstr: "not in the reporter's field",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := monsterBombField()
			env := newMonsterBombEnv(t, f)
			c.setup(t, env, f)

			env.dispatch()

			if env.selfDestructCalls != 0 {
				t.Fatalf("selfDestructCalls = %d, want 0", env.selfDestructCalls)
			}
			if !bytes.Contains(env.logs.Bytes(), []byte(c.wantSubstr)) {
				t.Errorf("log does not contain %q: %s", c.wantSubstr, env.logs.String())
			}
		})
	}
}

// TestMonsterBombDuplicateReportIsHarmless proves the handler does not try to
// dedupe reports itself — Registry.SelfDestruct on the atlas-monsters side is
// the exactly-once boundary (design D8), so two reports from the field just
// produce two requests.
func TestMonsterBombDuplicateReportIsHarmless(t *testing.T) {
	f := monsterBombField()
	env := newMonsterBombEnv(t, f)

	ten := tenant.MustFromContext(env.ctx)
	monster.GetLiveMirror().Put(ten, monsterBombMonsterId, monster.LiveEntry{Field: f, MonsterId: 8500003})

	env.dispatch()
	env.dispatch()

	if env.selfDestructCalls != 2 {
		t.Fatalf("selfDestructCalls = %d, want 2", env.selfDestructCalls)
	}
}
