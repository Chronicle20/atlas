package anniversary

import (
	"atlas-events/event/occurrence"
	"atlas-events/kafka/message/buff"
	"atlas-events/kafka/message/characterstatus"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// login builds a LOGIN status event for the given world/channel/character.
func login(worldId world.Id, channelId channel.Id, characterId uint32) characterstatus.StatusEvent[characterstatus.StatusEventLoginBody] {
	return characterstatus.StatusEvent[characterstatus.StatusEventLoginBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        characterstatus.StatusEventTypeLogin,
		Body:        characterstatus.StatusEventLoginBody{ChannelId: channelId},
	}
}

// seedCompletedOccurrence seeds a definition and one COMPLETED occurrence for
// it, so GetActiveByType (which only ever returns ACTIVE rows) finds
// nothing — the FR-A16 "nothing after completion" case.
func seedCompletedOccurrence(t *testing.T, db *gorm.DB, theType string, reason string) occurrence.Model {
	t.Helper()

	d := seedDefinition(t, db, theType)
	c, err := DecodeConfig(d.Configuration())
	must(t, err)

	oc := OccurrenceContext{
		ScheduledEnd:   c.ScheduledEnd,
		ExpMultiplier:  c.ExpMultiplier,
		DropMultiplier: c.DropMultiplier,
		BuffSourceId:   c.BuffSourceId,
	}
	raw, err := EncodeOccurrenceContext(oc)
	must(t, err)

	completedAt := time.Now()
	m, err := occurrence.NewBuilder(d.Id(), theType).
		SetState(occurrence.StateCompleted).
		SetContext(raw).
		SetConcurrencyKey(concurrencyKey).
		SetStartedAt(time.Now()).
		SetCompletedAt(&completedAt).
		SetCompletionReason(reason).
		Build()
	must(t, err)

	tn := tenant.MustFromContext(testCtx(t))
	entity, err := occurrence.ToEntity(m, tn.Id())
	must(t, err)
	must(t, db.Create(&entity).Error)

	made, err := occurrence.Make(entity)
	must(t, err)
	return made
}

// emittedApply decodes every message captured on topic whose Type equals
// wantType as a buff.Command[buff.ApplyCommandBody].
func (f *emitCapture) emittedApply(topic topic.Token, wantType string) []buff.Command[buff.ApplyCommandBody] {
	f.t.Helper()
	var out []buff.Command[buff.ApplyCommandBody]
	for _, m := range emitted.Messages(string(topic)) {
		var c buff.Command[buff.ApplyCommandBody]
		if err := json.Unmarshal(m.Value, &c); err != nil {
			f.t.Fatalf("decode buff command: %v", err)
		}
		if c.Type != wantType {
			continue
		}
		out = append(out, c)
	}
	return out
}

// FR-A7: logging in while the occurrence is active grants the configured
// buff, with the occurrence id as its correlation (FR-A12) — the SAME
// string Advance's cancelByCorrelationCommandProvider cancels by (R34-6).
func TestLoginDuringAnActiveOccurrenceGrantsTheBuff(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	o := seedActiveOccurrence(t, db, TypeName)

	must(t, NewLoginProcessor(f, testCtx(t), db).OnLogin(login(world.Id(1), channel.Id(4), 42)))

	applies := f.emittedApply(buff.EnvCommandTopic, buff.CommandTypeApply)
	if len(applies) != 1 {
		t.Fatalf("emitted %d applies, want 1", len(applies))
	}
	a := applies[0]
	if a.CharacterId != 42 || a.WorldId != world.Id(1) || a.ChannelId != channel.Id(4) {
		t.Fatalf("targeted %d in %d/%d", a.CharacterId, a.WorldId, a.ChannelId)
	}
	if a.Body.CorrelationId != o.Id().String() {
		t.Fatalf("correlation = %q, want %q", a.Body.CorrelationId, o.Id().String())
	}
	// defaultSpec's ExpMultiplier/DropMultiplier are both 2.0, carried as
	// amount 200, per ConversionDirect (Task 8).
	want := map[string]int32{"EXP_BUFF_RATE": 200, "ITEM_UP_BY_ITEM": 200}
	got := map[string]int32{}
	for _, c := range a.Body.Changes {
		got[c.Type] = c.Amount
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("changes = %v, want %v", got, want)
	}
	// FR-A5: the occurrence is the authoritative fact. The buff must not
	// carry a duration derived from the window — it is cancelled explicitly
	// at completion.
	if !a.Body.NoExpiry || a.Body.Duration != 0 {
		t.Fatalf("noExpiry=%v duration=%d, want true/0", a.Body.NoExpiry, a.Body.Duration)
	}
}

// FR-A16: after completion, a newly logging-in character gets nothing —
// GetActiveByType only ever returns ACTIVE rows.
func TestLoginAfterCompletionGrantsNothing(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	seedCompletedOccurrence(t, db, TypeName, ReasonScheduledEnd)

	must(t, NewLoginProcessor(f, testCtx(t), db).OnLogin(login(world.Id(1), channel.Id(4), 42)))

	if got := len(f.emittedApply(buff.EnvCommandTopic, buff.CommandTypeApply)); got != 0 {
		t.Fatalf("emitted %d applies after completion, want 0", got)
	}
}

// FR-A1: the multipliers are configuration. A 1.5x configuration produces
// amount 150, not a hardcoded 200.
func TestConfiguredMultipliersAreCarriedVerbatim(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	multiplier := func(v float64) defOpt {
		return func(s *testDefSpec) {
			s.cfg.ExpMultiplier = v
			s.cfg.DropMultiplier = v
		}
	}
	seedActiveOccurrence(t, db, TypeName, multiplier(1.5))

	must(t, NewLoginProcessor(f, testCtx(t), db).OnLogin(login(world.Id(1), channel.Id(4), 42)))

	applies := f.emittedApply(buff.EnvCommandTopic, buff.CommandTypeApply)
	if len(applies) != 1 {
		t.Fatalf("emitted %d applies, want 1", len(applies))
	}
	want := map[string]int32{"EXP_BUFF_RATE": 150, "ITEM_UP_BY_ITEM": 150}
	got := map[string]int32{}
	for _, c := range applies[0].Body.Changes {
		got[c.Type] = c.Amount
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("changes = %v, want %v", got, want)
	}
}

// A login with no Anniversary occurrence at all is a cheap no-op: no
// definition seeded at all means GetActiveByType returns an empty slice.
func TestLoginWithNoOccurrenceIsANoOp(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)

	must(t, NewLoginProcessor(f, testCtx(t), db).OnLogin(login(world.Id(1), channel.Id(4), 42)))

	if got := len(f.emittedApply(buff.EnvCommandTopic, buff.CommandTypeApply)); got != 0 {
		t.Fatalf("emitted %d applies with no occurrence, want 0", got)
	}
}
