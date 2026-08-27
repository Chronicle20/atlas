package asset

import (
	"atlas-channel/character"
	"atlas-channel/equipment"
	"atlas-channel/ring"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// countingRingProcessor is a test double satisfying ring.Processor whose
// GetRingSet counts its own invocations. Populate/Invalidate are never
// expected to be called from updateAppearance and panic if they are.
type countingRingProcessor struct {
	calls int32
}

func (c *countingRingProcessor) GetRingSet(_ uint32, _ equipment.Model) packetmodel.RingSet {
	atomic.AddInt32(&c.calls, 1)
	return packetmodel.RingSet{}
}

func (c *countingRingProcessor) GetRingRecords(_ uint32) packetmodel.RingRecords {
	panic("GetRingRecords: unexpected call from updateAppearance")
}

func (c *countingRingProcessor) Invalidate(_ uint32) {
	panic("Invalidate: unexpected call from updateAppearance")
}

func (c *countingRingProcessor) Populate(_ uint32) error {
	panic("Populate: unexpected call from updateAppearance")
}

var _ ring.Processor = (*countingRingProcessor)(nil)

// alwaysErrorProducer is a writer.Producer that fails before any encoder
// runs, so the test never needs a live socket connection -- session.Announce
// returns on the writerProducer error before touching the session's conn.
func alwaysErrorProducer(_ string) (writer.BodyFunc, error) {
	return nil, errors.New("no writer in test")
}

// TestUpdateAppearanceResolvesRingsOnce is the PRD §8 hot-path guard: the
// RingSet for the broadcasting character must be resolved once, outside the
// per-session closure ForSessionsInMap invokes -- not once per recipient
// session in the map. It fails loudly if a later refactor moves the
// ring.Processor call inside the per-session Operator.
func TestUpdateAppearanceResolvesRingsOnce(t *testing.T) {
	dbl := &countingRingProcessor{}
	prev := ringProcessorFn
	ringProcessorFn = func(_ logrus.FieldLogger, _ context.Context) ring.Processor { return dbl }
	t.Cleanup(func() { ringProcessorFn = prev })

	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tn)
	l := logrus.New()

	c := character.NewModelBuilder().SetId(1).SetSp("0").MustBuild()

	op := updateAppearance(l)(ctx)(alwaysErrorProducer)(c)

	for i := 0; i < 3; i++ {
		s := session.NewSession(uuid.New(), tn, 0, nil)
		_ = op(s)
	}

	if got := atomic.LoadInt32(&dbl.calls); got != 1 {
		t.Fatalf("GetRingSet called %d times across 3 recipient sessions, want 1", got)
	}
}
