package events

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor reads the atlas-events map-entry visuals projection (Task 16):
// what visual, if any, is already active in a map. This is a REST read, not
// a Kafka consumer — it answers "what should I draw right now" for a
// character entering the map mid-event.
type Processor interface {
	ActiveVisualsInMap(f field.Model) ([]RestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// ActiveVisualsInMap returns the transport error to the caller unchanged
// (FR-B16, FR-N15): an unreachable atlas-events must cost the visual, not
// abort map entry. The caller (SpawnForSelf) is responsible for logging and
// moving on.
func (p *ProcessorImpl) ActiveVisualsInMap(f field.Model) ([]RestModel, error) {
	return requests.SliceProvider[RestModel, RestModel](p.l, p.ctx)(requestVisualsInMap(f), Extract, model.Filters[RestModel]())()
}

func Extract(m RestModel) (RestModel, error) {
	return m, nil
}
