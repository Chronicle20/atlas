package mist

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	mistKafka "atlas-maps/kafka/message/mist"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// recordingProducer captures emitted messages by topic for assertions
// without going through Kafka.
type recordingProducer struct {
	mu       sync.Mutex
	messages map[string][]kafka.Message
}

func newRecordingProducer() *recordingProducer {
	return &recordingProducer{messages: map[string][]kafka.Message{}}
}

func (m *recordingProducer) Provider() producer.Provider {
	return func(token string) producer.MessageProducer {
		return func(prov model.Provider[[]kafka.Message]) error {
			msgs, err := prov()
			if err != nil {
				return err
			}
			m.mu.Lock()
			defer m.mu.Unlock()
			m.messages[token] = append(m.messages[token], msgs...)
			return nil
		}
	}
}

func (m *recordingProducer) Messages(topic string) []kafka.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]kafka.Message(nil), m.messages[topic]...)
}

func newTestMistProcessor(t *testing.T, tt tenant.Model, rec *recordingProducer) (*ProcessorImpl, context.Context) {
	t.Helper()
	logger, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), tt)
	return &ProcessorImpl{
		l:   logger,
		ctx: ctx,
		t:   tt,
		p:   rec.Provider(),
		r:   newTestMistRegistry(),
	}, ctx
}

func TestProcessor_Create_AddsToRegistryAndEmitsCreated(t *testing.T) {
	tt := mkRegTenant()
	rec := newRecordingProducer()
	p, _ := newTestMistProcessor(t, tt, rec)

	body := mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType: "MONSTER", OwnerId: 9001,
		OriginX: 100, OriginY: 200,
		LtX: -50, LtY: -30, RbX: 50, RbY: 30,
		Disease: "POISON", DiseaseValue: 80, DiseaseDuration: 30000,
		Duration: 10000, TickIntervalMs: 1000,
		SourceSkillId: 100020, SourceSkillLevel: 5,
	}

	m, err := p.Create(body)
	require.NoError(t, err)
	require.Equal(t, "POISON", m.Disease())
	require.Equal(t, uint32(9001), m.OwnerId())
	require.Equal(t, "MONSTER", m.OwnerType())

	// Registry side: mist is present under tenant.
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	got := p.r.GetByField(tt, f)
	require.Len(t, got, 1)
	require.Equal(t, m.Id(), got[0].Id())

	// Producer side: a single MIST_CREATED event was emitted.
	msgs := rec.Messages(mistKafka.EnvEventTopic)
	require.Len(t, msgs, 1, "expected exactly one MIST_CREATED message")

	var event mistKafka.Event[mistKafka.CreatedBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &event))
	require.Equal(t, mistKafka.EventTypeCreated, event.Type)
	require.Equal(t, m.Id(), event.MistId)
	require.Equal(t, tt.Id(), event.Tenant)
	require.Equal(t, int64(10000), event.Body.Duration)
	require.Equal(t, "MONSTER", event.Body.OwnerType)
	require.Equal(t, uint32(9001), event.Body.OwnerId)
	require.Equal(t, int16(100), event.Body.OriginX)
	require.Equal(t, int16(-50), event.Body.LtX)
	require.Equal(t, int16(50), event.Body.RbX)
}

func TestProcessor_Destroy_RemovesAndEmitsDestroyed(t *testing.T) {
	tt := mkRegTenant()
	rec := newRecordingProducer()
	p, _ := newTestMistProcessor(t, tt, rec)

	body := mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		Disease: "POISON", Duration: 10000, TickIntervalMs: 1000,
	}
	m, err := p.Create(body)
	require.NoError(t, err)

	removed, err := p.Destroy(m.Id(), mistKafka.ReasonExpired)
	require.NoError(t, err)
	require.Equal(t, m.Id(), removed.Id())

	// Registry side: mist is gone.
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	require.Empty(t, p.r.GetByField(tt, f))

	// Producer side: one MIST_CREATED followed by one MIST_DESTROYED.
	msgs := rec.Messages(mistKafka.EnvEventTopic)
	require.Len(t, msgs, 2)

	var destroyed mistKafka.Event[mistKafka.DestroyedBody]
	require.NoError(t, json.Unmarshal(msgs[1].Value, &destroyed))
	require.Equal(t, mistKafka.EventTypeDestroyed, destroyed.Type)
	require.Equal(t, mistKafka.ReasonExpired, destroyed.Body.Reason)
	require.Equal(t, m.Id(), destroyed.MistId)
}

func TestProcessor_Destroy_NotFound_ReturnsError(t *testing.T) {
	tt := mkRegTenant()
	rec := newRecordingProducer()
	p, _ := newTestMistProcessor(t, tt, rec)

	_, err := p.Destroy(uuid.New(), mistKafka.ReasonCancelled)
	require.Error(t, err)
	require.Empty(t, rec.Messages(mistKafka.EnvEventTopic))
}

func TestCreatedEventProvider_BuildsCreatedEvent(t *testing.T) {
	tt := mkRegTenant()
	id := uuid.New()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	m := NewBuilder(id, f).
		SetOwner("MONSTER", 9001).
		SetOrigin(100, 200).
		SetBounds(-50, -30, 50, 30).
		SetDuration(10 * time.Second).
		Build()

	prov := createdEventProvider(tt, m)
	require.NotNil(t, prov)
	msgs, err := prov()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var event mistKafka.Event[mistKafka.CreatedBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &event))
	require.Equal(t, mistKafka.EventTypeCreated, event.Type)
	require.Equal(t, id, event.MistId)
	require.Equal(t, tt.Id(), event.Tenant)
	require.Equal(t, int16(100), event.Body.OriginX)
	require.Equal(t, int64(10000), event.Body.Duration)
}

func TestDestroyedEventProvider_BuildsDestroyedEvent(t *testing.T) {
	tt := mkRegTenant()
	id := uuid.New()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	m := NewBuilder(id, f).Build()
	prov := destroyedEventProvider(tt, m, mistKafka.ReasonExpired)
	require.NotNil(t, prov)
	msgs, err := prov()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var event mistKafka.Event[mistKafka.DestroyedBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &event))
	require.Equal(t, mistKafka.EventTypeDestroyed, event.Type)
	require.Equal(t, mistKafka.ReasonExpired, event.Body.Reason)
	require.Equal(t, id, event.MistId)
}

// TestProcessor_Create_EmptyKinds_NormalizeToCharacterDisease pins FR-2.3: the
// existing atlas-monsters AREA_POISON producer omits the descriptors, and its
// behavior must be unchanged.
func TestProcessor_Create_EmptyKinds_NormalizeToCharacterDisease(t *testing.T) {
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := newTestMistProcessor(t, tt, newRecordingProducer())

	m, err := p.Create(mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType: "MONSTER", OwnerId: 7,
		Duration: 5000, TickIntervalMs: 1000,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if m.TargetKind() != mistKafka.TargetKindCharacter {
		t.Fatalf("TargetKind = %q, want CHARACTER", m.TargetKind())
	}
	if m.EffectKind() != mistKafka.EffectKindDisease {
		t.Fatalf("EffectKind = %q, want DISEASE", m.EffectKind())
	}
}

// TestProcessor_Create_ExplicitKinds_RoundTrip pins the player-cast path.
func TestProcessor_Create_ExplicitKinds_RoundTrip(t *testing.T) {
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := newTestMistProcessor(t, tt, newRecordingProducer())

	m, err := p.Create(mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType: "CHARACTER", OwnerId: 1001,
		TargetKind: mistKafka.TargetKindMonster,
		EffectKind: mistKafka.EffectKindDamageOverTime,
		Duration:   4000, TickIntervalMs: 1000,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if m.TargetKind() != mistKafka.TargetKindMonster || m.EffectKind() != mistKafka.EffectKindDamageOverTime {
		t.Fatalf("kinds = (%q,%q), want (MONSTER,DAMAGE_OVER_TIME)", m.TargetKind(), m.EffectKind())
	}
}

// TestProcessor_Create_DerivesAffectedAreaTypeFromOwner pins the wire value
// that made Poison Mist kill its own caster (task-200 live test).
//
// AFFECTEDAREA::nType == 0 is the client's MOB-disease-cloud marker: v83
// CAffectedAreaPool::GetAffectedAreaByPoint (sub_431783) selects an area for
// the LOCAL USER iff `!nType && tCur >= tStart && PtInRect(rcArea, ptUser)`,
// and CUserLocal::Update (@0x94b801) then hits the user for
// `AFFECTEDAREA.nDamage * (100 - resist) / 100`. nDamage is written ONLY on
// the mob-skill arms (nSkillID 130/131) of AffectedAreaAnimationCreated, never
// on the 2111003 arm -- so a character-owned mist sent as nType 0 billed the
// caster an uninitialized value (observed: 1434803, clamped to 999999, dead).
//
// Both arms are asserted. The MONSTER arm is not symmetry for its own sake: 0
// is what makes the client apply a mob's cloud to players standing in it, and
// that pre-task-200 AREA_POISON behaviour must not change.
func TestProcessor_Create_DerivesAffectedAreaTypeFromOwner(t *testing.T) {
	cases := []struct {
		name      string
		ownerType string
		want      int32
	}{
		{"character-owned mist is a user skill area", "CHARACTER", AffectedAreaTypeUserSkill},
		{"monster-owned mist stays a mob disease cloud", "MONSTER", AffectedAreaTypeMobSkill},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tt := mkRegTenant()
			rec := newRecordingProducer()
			p, _ := newTestMistProcessor(t, tt, rec)

			m, err := p.Create(mistKafka.CreateCommandBody{
				WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
				OwnerType: tc.ownerType, OwnerId: 1,
				OriginX: 100, OriginY: 200,
				LtX: -50, LtY: -30, RbX: 50, RbY: 30,
				Disease: "POISON", DiseaseDuration: 30000,
				Duration: 10000, TickIntervalMs: 1000,
				SourceSkillId: 2111003, SourceSkillLevel: 30,
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, m.Type())

			// The value has to reach the wire, not just the registry entry --
			// atlas-channel copies CreatedBody.Type straight into the
			// AffectedAreaCreated packet's nType.
			msgs := rec.Messages(mistKafka.EnvEventTopic)
			require.Len(t, msgs, 1)
			var event mistKafka.Event[mistKafka.CreatedBody]
			require.NoError(t, json.Unmarshal(msgs[0].Value, &event))
			require.Equal(t, tc.want, event.Body.Type)
		})
	}
}

// FR-2.5: an unrecognised kind must be REJECTED, not silently normalised to
// DISEASE -- a mist that applies the wrong effect to the wrong targets is
// worse than no mist.
func TestCreate_UnknownEffectKind_RejectedAndNoMist(t *testing.T) {
	tt := mkRegTenant()
	rec := newRecordingProducer()
	p, _ := newTestMistProcessor(t, tt, rec)

	_, err := p.Create(mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType:  OwnerTypeCharacter,
		OwnerId:    1001,
		TargetKind: mistKafka.TargetKindCharacter,
		EffectKind: "TELEPORT_EVERYONE",
		Duration:   30000,
	})

	require.ErrorIs(t, err, ErrUnknownKind)
	require.Empty(t, p.r.AllByTenant(tt))
	require.Empty(t, rec.Messages(mistKafka.EnvEventTopic))
}

func TestCreate_UnknownTargetKind_RejectedAndNoMist(t *testing.T) {
	tt := mkRegTenant()
	rec := newRecordingProducer()
	p, _ := newTestMistProcessor(t, tt, rec)

	_, err := p.Create(mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType:  OwnerTypeCharacter,
		TargetKind: "NPC",
		EffectKind: mistKafka.EffectKindDisease,
		Duration:   30000,
	})

	require.ErrorIs(t, err, ErrUnknownKind)
	require.Empty(t, p.r.AllByTenant(tt))
}

// FR-2.3: the pre-task-200 atlas-monsters AREA_POISON producer sends neither
// kind. That must keep working, unchanged.
func TestCreate_EmptyKinds_NormalizeToCharacterDisease(t *testing.T) {
	tt := mkRegTenant()
	rec := newRecordingProducer()
	p, _ := newTestMistProcessor(t, tt, rec)

	m, err := p.Create(mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType: OwnerTypeMonster,
		Duration:  30000,
	})

	require.NoError(t, err)
	require.Equal(t, mistKafka.TargetKindCharacter, m.TargetKind())
	require.Equal(t, mistKafka.EffectKindDisease, m.EffectKind())
}

// The recovery magnitude and the party snapshot must survive the command ->
// model hop; a dropped setter heals nobody and fails silently.
func TestCreate_CarriesRecoveryFields(t *testing.T) {
	tt := mkRegTenant()
	rec := newRecordingProducer()
	p, _ := newTestMistProcessor(t, tt, rec)

	m, err := p.Create(mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType:      OwnerTypeCharacter,
		OwnerId:        1001,
		TargetKind:     mistKafka.TargetKindCharacter,
		EffectKind:     mistKafka.EffectKindRecovery,
		RecoveryMp:     38,
		PartyMemberIds: []uint32{1001, 1002},
		Duration:       30000,
		TickIntervalMs: 3000,
	})

	require.NoError(t, err)
	require.Equal(t, int32(38), m.RecoveryMp())
	require.Equal(t, []uint32{1001, 1002}, m.PartyMemberIds())
}
