package handler

import (
	cashData "atlas-channel/data/cash"
	sagaMsg "atlas-channel/kafka/message/saga"
	"atlas-channel/saga"
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// installCashItemDataSeam swaps the cash-data lookup for the test and returns a
// restore func (same package-var injection precedent as installCashItemInSlotSeam).
func installCashItemDataSeam(t *testing.T, meso uint32, err error) func() {
	t.Helper()
	orig := cashItemDataFunc
	cashItemDataFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (cashData.RestModel, error) {
		if err != nil {
			return cashData.RestModel{}, err
		}
		return cashData.RestModel{Meso: meso}, nil
	}
	return func() { cashItemDataFunc = orig }
}

// The saga shape is the FR-4 invariant: destroy-first, exactly two steps, the
// award carrying ShowEffect so the meso gain renders as the chat line.
func TestBuildMesoSackUseSaga(t *testing.T) {
	txn := uuid.New()
	now := time.Now()
	s := buildMesoSackUseSaga(txn, now, 4242, item.Id(5200000), world.Id(0), channel.Id(1), 1000000)

	if s.TransactionId != txn {
		t.Errorf("transactionId: got %s, want %s", s.TransactionId, txn)
	}
	if s.SagaType != saga.MesoSackUse {
		t.Errorf("sagaType: got %s, want %s", s.SagaType, saga.MesoSackUse)
	}
	if len(s.Steps) != 2 {
		t.Fatalf("steps: got %d, want 2", len(s.Steps))
	}

	if s.Steps[0].StepId != "consume_meso_sack" {
		t.Errorf("step 1 id: got %q, want %q", s.Steps[0].StepId, "consume_meso_sack")
	}
	if s.Steps[0].Action != saga.DestroyAsset {
		t.Errorf("step 1 action: got %s, want %s (destroy-first is mandatory)", s.Steps[0].Action, saga.DestroyAsset)
	}
	dp, ok := s.Steps[0].Payload.(saga.DestroyAssetPayload)
	if !ok {
		t.Fatalf("step 1 payload type: %T", s.Steps[0].Payload)
	}
	if dp.CharacterId != 4242 || dp.TemplateId != 5200000 || dp.Quantity != 1 || dp.RemoveAll {
		t.Errorf("destroy payload mismatch: %+v", dp)
	}

	if s.Steps[1].StepId != "award_mesos" {
		t.Errorf("step 2 id: got %q, want %q", s.Steps[1].StepId, "award_mesos")
	}
	if s.Steps[1].Action != saga.AwardMesos {
		t.Errorf("step 2 action: got %s, want %s", s.Steps[1].Action, saga.AwardMesos)
	}
	ap, ok := s.Steps[1].Payload.(saga.AwardMesosPayload)
	if !ok {
		t.Fatalf("step 2 payload type: %T", s.Steps[1].Payload)
	}
	if ap.CharacterId != 4242 || ap.Amount != 1000000 || !ap.ShowEffect {
		t.Errorf("award payload mismatch: %+v", ap)
	}
	if ap.ActorId != 5200000 || ap.ActorType != "ITEM" {
		t.Errorf("actor mismatch: got %d/%q, want 5200000/ITEM", ap.ActorId, ap.ActorType)
	}
	if ap.WorldId != world.Id(0) || ap.ChannelId != channel.Id(1) {
		t.Errorf("field mismatch: got %d/%d, want 0/1", ap.WorldId, ap.ChannelId)
	}
}

// A resolvable, in-range amount creates exactly one saga and announces nothing:
// the success unlock rides on atlas-character's STAT_CHANGED{Meso}, which
// already carries ExclRequestSent. An extra empty StatChanged here would race it.
func TestMesoSackUseCreatesSagaAndAnnouncesNothing(t *testing.T) {
	const characterId = uint32(4242)
	const itemId = uint32(5200000)

	restoreData := installCashItemDataSeam(t, 1000000, nil)
	defer restoreData()
	captured, restoreProducer := installCapturingProducer()
	defer restoreProducer()

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	handleMesoSackUse(logrus.New(), ctx, rec.producer())(s, item.Id(itemId))

	if got := len((*captured)[sagaMsg.EnvCommandTopic]); got != 1 {
		t.Fatalf("emitted %d saga commands, want exactly 1", got)
	}
	if rec.calls != 0 {
		t.Fatalf("announced %d packets on the success path, want 0", rec.calls)
	}
}

// Fail closed: a zero amount (Maple Point sack 5200009, or a tenant whose WZ was
// not re-ingested) must consume nothing, award nothing, and unlock the client.
// Same for an amount past the int32 ceiling of AwardMesosPayload.Amount — without
// that guard a large sack wraps negative and TAKES mesos — and for a lookup error.
func TestMesoSackUseRejectsAndUnlocks(t *testing.T) {
	cases := []struct {
		name string
		meso uint32
		err  error
	}{
		{"zero amount (maple point sack)", 0, nil},
		{"above int32 ceiling", uint32(math.MaxInt32) + 1, nil},
		{"cash data lookup failure", 0, errors.New("404")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restoreData := installCashItemDataSeam(t, tc.meso, tc.err)
			defer restoreData()
			captured, restoreProducer := installCapturingProducer()
			defer restoreProducer()

			s, ctx, cleanup := newCashItemUseTestSession(t, 4242)
			defer cleanup()

			rec := &gaugeProducerRecorder{}
			handleMesoSackUse(logrus.New(), ctx, rec.producer())(s, item.Id(5200009))

			if got := len((*captured)[sagaMsg.EnvCommandTopic]); got != 0 {
				t.Fatalf("emitted %d saga commands, want 0 (nothing may be consumed)", got)
			}
			if rec.calls != 1 {
				t.Fatalf("announced %d packets, want exactly 1 (the enable-actions unlock)", rec.calls)
			}
		})
	}
}

// The constant must stay 19 for every version: Atlas derives the type from the
// server-resolved template id and the type never rides the wire, so the v48 (17)
// and v61 (18) client tables are irrelevant. A version gate here would break the
// branch on those builds (design §3.1(a)).
func TestCurrencySackTypeIsNineteenOnEveryVersion(t *testing.T) {
	for _, v := range []struct {
		region string
		major  uint16
		minor  uint16
	}{
		{"GMS", 48, 1},
		{"GMS", 61, 1},
		{"GMS", 72, 1},
		{"GMS", 79, 1},
		{"GMS", 83, 1},
		{"GMS", 84, 1},
		{"GMS", 87, 1},
		{"GMS", 92, 1},
		{"GMS", 95, 1},
		{"JMS", 185, 1},
	} {
		ten := mustTenant(t, v.region, v.major, v.minor)
		for _, id := range []uint32{5200000, 5200001, 5200002, 5202000} {
			if got := GetCashSlotItemType(ten)(item.Id(id)); got != CashSlotItemTypeCurrencySack {
				t.Errorf("%s v%d: GetCashSlotItemType(%d) = %d, want %d", v.region, v.major, id, got, CashSlotItemTypeCurrencySack)
			}
		}
	}
}
