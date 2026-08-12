package tasks

import (
	"atlas-maps/mist"
	"context"
	"encoding/json"
	"testing"
	"time"

	mistKafka "atlas-maps/kafka/message/mist"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// recoveryMist builds a live RECOVERY mist at (100,100) whose rect covers
// (0,0)..(200,200), scoped to the given party snapshot.
func recoveryMist(t *testing.T, f field.Model, party []uint32) mist.Mist {
	t.Helper()
	return mist.NewBuilder(uuid.New(), f).
		SetOwner(mist.OwnerTypeCharacter, party[0]).
		SetOrigin(100, 100).
		SetBounds(-100, -100, 100, 100).
		SetKinds(mistKafka.TargetKindCharacter, mistKafka.EffectKindRecovery).
		SetRecovery(38, party).
		SetDuration(30 * time.Second).
		SetTickInterval(3 * time.Second).
		Build()
}

// decodeChangeMp returns the (characterId, amount) pairs the tick emitted on
// COMMAND_TOPIC_CHARACTER, so the test asserts the wire shape rather than an
// internal call.
func decodeChangeMp(t *testing.T, rec *recordingProducer) []struct {
	CharacterId uint32
	Amount      int16
} {
	t.Helper()
	var out []struct {
		CharacterId uint32
		Amount      int16
	}
	for _, m := range rec.MessagesOn(EnvCommandTopicCharacter) {
		var env struct {
			CharacterId uint32 `json:"characterId"`
			Type        string `json:"type"`
			Body        struct {
				Amount int16 `json:"amount"`
			} `json:"body"`
		}
		require.NoError(t, json.Unmarshal(m.Value, &env))
		require.Equal(t, "CHANGE_MP", env.Type)
		out = append(out, struct {
			CharacterId uint32
			Amount      int16
		}{env.CharacterId, env.Body.Amount})
	}
	return out
}

func TestTickRecovery_HealsPartyMembersInsideOnly(t *testing.T) {
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	f := field.NewBuilder(0, 0, 100000000).Build()
	tt := mkTickTenant()

	// 1001 caster inside; 1002 party inside; 1003 NON-party inside;
	// 1004 party but OUTSIDE the rect.
	m := recoveryMist(t, f, []uint32{1001, 1002, 1004})
	require.NoError(t, reg.Add(tt, m))

	mt := newTestMistTick(t, reg, rec, func(_ context.Context, cid uint32) (int16, int16, uint16, error) {
		if cid == 1004 {
			return 900, 900, 500, nil
		}
		return 100, 100, 500, nil
	})
	mt.charsInField = func(tenant.Model, field.Model) []uint32 { return []uint32{1001, 1002, 1003, 1004} }

	mt.runOnce(context.Background())

	got := decodeChangeMp(t, rec)
	require.Len(t, got, 2)
	require.ElementsMatch(t, []uint32{1001, 1002}, []uint32{got[0].CharacterId, got[1].CharacterId})
	require.Equal(t, int16(38), got[0].Amount)
	require.Equal(t, int16(38), got[1].Amount)
}

// FR-5.3: a dead character in the cloud is not healed. atlas-character's
// ChangeMP clamps to max MP but does not check HP, so the check lives here.
func TestTickRecovery_SkipsDeadCharacter(t *testing.T) {
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	f := field.NewBuilder(0, 0, 100000000).Build()
	tt := mkTickTenant()
	require.NoError(t, reg.Add(tt, recoveryMist(t, f, []uint32{1001, 1002})))

	mt := newTestMistTick(t, reg, rec, func(_ context.Context, cid uint32) (int16, int16, uint16, error) {
		if cid == 1002 {
			return 100, 100, 0, nil
		}
		return 100, 100, 500, nil
	})
	mt.charsInField = func(tenant.Model, field.Model) []uint32 { return []uint32{1001, 1002} }

	mt.runOnce(context.Background())

	got := decodeChangeMp(t, rec)
	require.Len(t, got, 1)
	require.Equal(t, uint32(1001), got[0].CharacterId)
}

// FR-2.5 defence in depth: Processor.Create already rejects unknown kinds, so
// a mist built directly (a test, or a future producer) must not silently fall
// through to the DISEASE arm and disease everyone in the cloud.
func TestTickOneMist_UnknownEffectKind_WarnsAndEmitsNothing(t *testing.T) {
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	f := field.NewBuilder(0, 0, 100000000).Build()
	tt := mkTickTenant()

	m := mist.NewBuilder(uuid.New(), f).
		SetOwner(mist.OwnerTypeCharacter, 1001).
		SetOrigin(100, 100).
		SetBounds(-100, -100, 100, 100).
		SetKinds(mistKafka.TargetKindCharacter, "TELEPORT_EVERYONE").
		SetDuration(30 * time.Second).
		SetTickInterval(3 * time.Second).
		Build()
	require.NoError(t, reg.Add(tt, m))

	mt := newTestMistTick(t, reg, rec, func(context.Context, uint32) (int16, int16, uint16, error) {
		return 100, 100, 500, nil
	})
	mt.charsInField = func(tenant.Model, field.Model) []uint32 { return []uint32{1001} }
	hook := attachLogHook(t, mt)

	mt.runOnce(context.Background())

	require.Empty(t, rec.AllMessages())
	require.Contains(t, lastMessage(hook), "TELEPORT_EVERYONE")
}

// A PROTECTION mist is created with tickInterval 0, so ShouldTick is false
// and it never reaches the effect-kind switch -- it only expires. Pinned so a
// future non-zero interval cannot make it disease its own party.
func TestTickOneMist_ProtectionMistDoesNotTick(t *testing.T) {
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	f := field.NewBuilder(0, 0, 100000000).Build()
	tt := mkTickTenant()

	m := mist.NewBuilder(uuid.New(), f).
		SetOwner(mist.OwnerTypeCharacter, 1001).
		SetOrigin(100, 100).
		SetBounds(-100, -100, 100, 100).
		SetKinds(mistKafka.TargetKindCharacter, mistKafka.EffectKindProtection).
		SetDuration(31 * time.Second).
		SetTickInterval(0).
		Build()
	require.NoError(t, reg.Add(tt, m))

	mt := newTestMistTick(t, reg, rec, func(context.Context, uint32) (int16, int16, uint16, error) {
		return 100, 100, 500, nil
	})
	mt.charsInField = func(tenant.Model, field.Model) []uint32 { return []uint32{1001} }

	mt.runOnce(context.Background())

	require.Empty(t, rec.AllMessages())
}
