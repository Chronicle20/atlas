package jobavailability

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	// GetAvailable returns every job identity released/playable at the
	// requesting tenant's (region, major, minor), sorted ascending by that
	// version's wire id (constants/job.Set.AvailableIdentities' ordering).
	GetAvailable() []RestModel
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetAvailable() []RestModel {
	t := tenant.MustFromContext(p.ctx)
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())

	identities := set.Job.AvailableIdentities()
	ms := make([]RestModel, 0, len(identities))
	for _, id := range identities {
		wire, ok := set.Job.Wire(id)
		if !ok {
			// Invariant: AvailableIdentities is a subset of this version's
			// semantics (Set.Resolve/Wire), so every available identity has
			// a wire binding. Skip defensively rather than emit a bogus id.
			p.l.Errorf("Available job identity [%d] has no wire binding for this tenant version; skipping.", id)
			continue
		}
		m := RestModel{Id: uint16(wire), Name: set.Job.Name(id), Identity: uint16(id)}
		if pw, ok := set.Job.ParentWire(id); ok {
			// Copy into a local: &pw would alias the loop-scoped value.
			parent := uint16(pw)
			m.Parent = &parent
		}
		ms = append(ms, m)
	}
	return ms
}
