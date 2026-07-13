package service

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

// Projection is the two-method surface Bootstrap drives for opt-in
// configuration-projection wiring (design D6). Full wiring in Task 5.
type Projection interface {
	Start(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error
	WaitCaughtUp(ctx context.Context) error
}

type projectionConfig struct {
	baseGroupId string
	build       ProjectionBuilder
}

// ProjectionBuilder builds the service's Projection from the resolved topics.
type ProjectionBuilder func(t ProjectionTopics) Projection

// ProjectionTopics carries the env-resolved config-status topic names.
type ProjectionTopics struct {
	ServiceStatus string
	TenantStatus  string
}

func (r *Runtime) startProjection(pc *projectionConfig) {
	panic("projection wiring lands in Task 5; no caller passes WithConfigProjection yet")
}
