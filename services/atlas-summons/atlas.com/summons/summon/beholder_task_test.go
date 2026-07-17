package summon

import (
	buffmsg "atlas-summons/buff"
	charmsg "atlas-summons/character"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// capture records the topic and decoded payload of every emitted message so the
// test can assert the cross-service contracts without a live kafka broker.
type beholderCapturedMessage struct {
	topic   string
	payload []byte
}

type beholderCaptureEmitter struct {
	mu  sync.Mutex
	msg []beholderCapturedMessage
}

func (c *beholderCaptureEmitter) emit(_ context.Context, topic string, provider model.Provider[[]kafka.Message]) error {
	msgs, err := provider()
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range msgs {
		c.msg = append(c.msg, beholderCapturedMessage{topic: topic, payload: m.Value})
	}
	return nil
}

func (c *beholderCaptureEmitter) byTopic(topic string) []beholderCapturedMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]beholderCapturedMessage, 0)
	for _, m := range c.msg {
		if m.topic == topic {
			out = append(out, m)
		}
	}
	return out
}

func newBeholderModel(id uint32, owner uint32, f field.Model, nextHeal, nextBuff time.Time) Model {
	return NewBuilder().SetId(id).SetOwnerCharacterId(owner).SetField(f).
		SetSummonType(SummonTypeBuffAura).SetMovementType(MovementFollow).
		SetSkillId(1320009).SetSkillLevel(10).
		SetNextHealAt(nextHeal).SetNextBuffAt(nextBuff).
		SetHealAmount(60).
		SetHealInterval(4 * time.Second).
		SetBuffInterval(4 * time.Second).
		SetBuffSourceId(int32(1320009)).
		SetBuffLevel(10).
		SetBuffDuration(213000).
		SetBuffChanges([]StatChange{{Type: "WATK", Amount: 30}}).
		Build()
}

func setupBeholderRegistry(t *testing.T) (tenant.Model, context.Context, field.Model) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	registry = newRegistry(rc)
	idAllocator = &IdAllocator{inner: objectid.NewRedisAllocator(rc)}

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	return ten, ctx, f
}

func TestBeholderSweepFiresHealAndBuffWhenDue(t *testing.T) {
	ten, ctx, f := setupBeholderRegistry(t)

	now := time.Now()
	due := newBeholderModel(2000001, 42, f, now.Add(-time.Second), now.Add(-time.Second))
	notDue := newBeholderModel(2000002, 43, f, now.Add(time.Hour), now.Add(time.Hour))

	if err := GetRegistry().Put(ctx, ten, due); err != nil {
		t.Fatal(err)
	}
	if err := GetRegistry().Put(ctx, ten, notDue); err != nil {
		t.Fatal(err)
	}

	cap := &beholderCaptureEmitter{}
	task := NewBeholderTask(logrus.New(), context.Background(), time.Second)
	task.emit = cap.emit
	task.pick = func(int) int { return 0 } // deterministic: pick the first pool stat
	task.Run()

	// CHANGE_HP assertion.
	hpMsgs := cap.byTopic(charmsg.EnvCommandTopic)
	if len(hpMsgs) != 1 {
		t.Fatalf("expected 1 CHANGE_HP message, got %d", len(hpMsgs))
	}
	var hp charmsg.Command[charmsg.ChangeHPBody]
	if err := json.Unmarshal(hpMsgs[0].payload, &hp); err != nil {
		t.Fatalf("decode CHANGE_HP: %v", err)
	}
	if hp.Type != charmsg.CommandChangeHP {
		t.Fatalf("expected type CHANGE_HP, got %q", hp.Type)
	}
	if hp.CharacterId != 42 {
		t.Fatalf("expected CharacterId 42, got %d", hp.CharacterId)
	}
	if hp.Body.Amount != 60 {
		t.Fatalf("expected Amount 60, got %d", hp.Body.Amount)
	}

	// buff APPLY assertion.
	buffMsgs := cap.byTopic(buffmsg.EnvCommandTopic)
	if len(buffMsgs) != 1 {
		t.Fatalf("expected 1 APPLY message, got %d", len(buffMsgs))
	}
	var ap buffmsg.Command[buffmsg.ApplyCommandBody]
	if err := json.Unmarshal(buffMsgs[0].payload, &ap); err != nil {
		t.Fatalf("decode APPLY: %v", err)
	}
	if ap.Type != buffmsg.CommandTypeApply {
		t.Fatalf("expected type APPLY, got %q", ap.Type)
	}
	if ap.CharacterId != 42 {
		t.Fatalf("expected CharacterId 42, got %d", ap.CharacterId)
	}
	if ap.Body.FromId != 42 {
		t.Fatalf("expected FromId 42, got %d", ap.Body.FromId)
	}
	if ap.Body.SourceId != int32(1320009) {
		t.Fatalf("expected SourceId %d, got %d", int32(1320009), ap.Body.SourceId)
	}
	if !ap.Body.Accumulate {
		t.Fatalf("expected Accumulate=true so Hex stats accumulate per-stat")
	}
	if len(ap.Body.Changes) != 1 {
		t.Fatalf("expected exactly 1 stat per pulse (one random buff), got %d", len(ap.Body.Changes))
	}
	if ap.Body.Changes[0].Type != "WATK" {
		t.Fatalf("expected the picked stat WATK, got %q", ap.Body.Changes[0].Type)
	}

	// SKILL pulse assertion: both the heal and the buff sweep emit a SummonSkill
	// status event so the channel plays the Beholder's cast animation map-wide.
	skillMsgs := cap.byTopic(EnvEventTopicSummonStatus)
	if len(skillMsgs) != 2 {
		t.Fatalf("expected 2 SKILL pulse messages (heal + buff), got %d", len(skillMsgs))
	}

	// Timers advanced and persisted.
	updated, err := GetRegistry().Get(ctx, ten, 2000001)
	if err != nil {
		t.Fatalf("re-fetch beholder: %v", err)
	}
	if !updated.NextHealAt().After(now) {
		t.Fatalf("expected NextHealAt advanced past now, got %v", updated.NextHealAt())
	}
	if !updated.NextBuffAt().After(now) {
		t.Fatalf("expected NextBuffAt advanced past now, got %v", updated.NextBuffAt())
	}
	// Timers serialize as unix-milli, so compare against the millisecond-truncated
	// original advanced by the interval.
	wantHeal := due.NextHealAt().Truncate(time.Millisecond).Add(4 * time.Second)
	if !updated.NextHealAt().Equal(wantHeal) {
		t.Fatalf("expected NextHealAt %v, got %v", wantHeal, updated.NextHealAt())
	}
	wantBuff := due.NextBuffAt().Truncate(time.Millisecond).Add(4 * time.Second)
	if !updated.NextBuffAt().Equal(wantBuff) {
		t.Fatalf("expected NextBuffAt %v, got %v", wantBuff, updated.NextBuffAt())
	}
}

func TestBeholderSweepSkipsWhenNotDue(t *testing.T) {
	ten, ctx, f := setupBeholderRegistry(t)

	now := time.Now()
	notDue := newBeholderModel(2000003, 44, f, now.Add(time.Hour), now.Add(time.Hour))
	if err := GetRegistry().Put(ctx, ten, notDue); err != nil {
		t.Fatal(err)
	}

	cap := &beholderCaptureEmitter{}
	task := NewBeholderTask(logrus.New(), context.Background(), time.Second)
	task.emit = cap.emit
	task.pick = func(int) int { return 0 } // deterministic: pick the first pool stat
	task.Run()

	if got := len(cap.byTopic(charmsg.EnvCommandTopic)); got != 0 {
		t.Fatalf("expected no CHANGE_HP messages, got %d", got)
	}
	if got := len(cap.byTopic(buffmsg.EnvCommandTopic)); got != 0 {
		t.Fatalf("expected no APPLY messages, got %d", got)
	}

	// Timers unchanged.
	unchanged, err := GetRegistry().Get(ctx, ten, 2000003)
	if err != nil {
		t.Fatalf("re-fetch beholder: %v", err)
	}
	if !unchanged.NextHealAt().Equal(notDue.NextHealAt().Truncate(time.Millisecond)) {
		t.Fatalf("expected NextHealAt unchanged, got %v", unchanged.NextHealAt())
	}
}

// Over successive pulses the buff sweep applies one random stat each, covering the
// whole pool (accumulation), and a re-rolled stat is re-applied (timer refresh).
func TestBeholderSweepBuffAccumulatesAcrossPulses(t *testing.T) {
	ten, ctx, f := setupBeholderRegistry(t)

	m := NewBuilder().SetId(2000010).SetOwnerCharacterId(50).SetField(f).
		SetSummonType(SummonTypeBuffAura).SetMovementType(MovementFollow).
		SetSkillId(1320009).SetSkillLevel(25).
		SetNextBuffAt(time.Now().Add(-time.Second)).
		SetBuffInterval(time.Second).
		SetBuffSourceId(int32(1320009)).SetBuffLevel(25).SetBuffDuration(99000).
		SetBuffChanges([]StatChange{{Type: "WDEF", Amount: 100}, {Type: "MDEF", Amount: 100}, {Type: "WATK", Amount: 15}}).
		Build()
	if err := GetRegistry().Put(ctx, ten, m); err != nil {
		t.Fatal(err)
	}

	cap := &beholderCaptureEmitter{}
	task := NewBeholderTask(logrus.New(), context.Background(), time.Second)
	task.emit = cap.emit
	seq := []int{0, 1, 2, 0} // WDEF, MDEF, WATK, WDEF (re-roll)
	i := 0
	task.pick = func(n int) int { v := seq[i] % n; i++; return v }

	base := time.Now()
	for p := 0; p < len(seq); p++ {
		cur, err := GetRegistry().Get(ctx, ten, 2000010)
		if err != nil {
			t.Fatalf("re-fetch: %v", err)
		}
		task.sweepBuff(ctx, ten, cur, base.Add(time.Duration(p+1)*time.Hour))
	}

	got := map[string]int{}
	for _, msg := range cap.byTopic(buffmsg.EnvCommandTopic) {
		var ap buffmsg.Command[buffmsg.ApplyCommandBody]
		if err := json.Unmarshal(msg.payload, &ap); err != nil {
			t.Fatalf("decode APPLY: %v", err)
		}
		if !ap.Body.Accumulate || len(ap.Body.Changes) != 1 {
			t.Fatalf("each pulse must carry exactly one stat with Accumulate=true, got %+v", ap.Body)
		}
		got[ap.Body.Changes[0].Type]++
	}

	for _, s := range []string{"WDEF", "MDEF", "WATK"} {
		if got[s] == 0 {
			t.Fatalf("expected stat %s applied across pulses, got %v", s, got)
		}
	}
	if got["WDEF"] != 2 {
		t.Fatalf("expected WDEF rolled twice (indices 0 and 3), got %d", got["WDEF"])
	}
}
