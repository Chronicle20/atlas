package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Projection is the two-method surface Bootstrap drives for opt-in
// configuration-projection wiring (design D6). Each projection service's
// existing Subscriber/CaughtUp pair satisfies it via ProjectionFuncs.
type Projection interface {
	Start(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error
	WaitCaughtUp(ctx context.Context) error
}

// ProjectionTopics carries the env-resolved config-status topic names.
type ProjectionTopics struct {
	ServiceStatus string
	TenantStatus  string
}

// ProjectionBuilder builds the service's Projection from the resolved topics.
type ProjectionBuilder func(t ProjectionTopics) Projection

// ProjectionFuncs adapts a service's Subscriber.Start / CaughtUp.WaitCaughtUp
// pair to the Projection interface without a per-service adapter type.
type ProjectionFuncs struct {
	StartFunc        func(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error
	WaitCaughtUpFunc func(ctx context.Context) error
}

func (p ProjectionFuncs) Start(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error {
	return p.StartFunc(ctx, l, wg, groupId)
}

func (p ProjectionFuncs) WaitCaughtUp(ctx context.Context) error {
	return p.WaitCaughtUpFunc(ctx)
}

type projectionConfig struct {
	baseGroupId string
	build       ProjectionBuilder
}

// WithConfigProjection makes Bootstrap read the config-status topic env
// vars, build the service's Projection, and start it bound to the teardown
// context/waitgroup under a per-process consumer group id (replaying the
// compacted log from FirstOffset on every container start). The catch-up
// gate stays an explicit rt.AwaitProjectionCatchUp() call so each service
// keeps its own gate position (world/character-factory gate after the REST
// server starts; login/channel gate before building listeners).
func WithConfigProjection(baseGroupId string, build ProjectionBuilder) Option {
	return func(c *bootstrapConfig) {
		c.projection = &projectionConfig{baseGroupId: baseGroupId, build: build}
	}
}

func (r *Runtime) startProjection(pc *projectionConfig) {
	topics := ProjectionTopics{
		ServiceStatus: os.Getenv("EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS"),
		TenantStatus:  os.Getenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS"),
	}
	if topics.TenantStatus == "" {
		r.logger.Warn("projection: EVENT_TOPIC_CONFIGURATION_TENANT_STATUS is not set; tenant config updates will not propagate live")
	}
	p := pc.build(topics)
	// Per-process group id so each container start replays the full
	// compacted log from FirstOffset; a shared group would resume from the
	// previous run's committed offset and leave the in-memory State empty.
	groupId := fmt.Sprintf("%s - projection - %s", pc.baseGroupId, uuid.New().String())
	if err := p.Start(r.tdm.Context(), r.logger, r.tdm.WaitGroup(), groupId); err != nil {
		r.logger.WithError(err).Fatal("Unable to start configuration projection subscriber.")
	}
	r.projection = p
}

// AwaitProjectionCatchUp blocks until the projection reports caught-up or
// the PROJECTION_CATCHUP_TIMEOUT_S window (default 5 minutes — covers
// fresh PR envs where atlas-pr-bootstrap is still writing initial configs)
// elapses, in which case it Fatals. Panics if Bootstrap was not given
// WithConfigProjection (programmer error, not a silent no-op).
func (r *Runtime) AwaitProjectionCatchUp() {
	if r.projection == nil {
		panic("service.Runtime.AwaitProjectionCatchUp called without WithConfigProjection")
	}
	ctx, cancel := context.WithTimeout(r.tdm.Context(), parseProjectionCatchupTimeout())
	defer cancel()
	if err := r.projection.WaitCaughtUp(ctx); err != nil {
		r.logger.WithError(err).Fatal("Configuration projection failed to catch up.")
	}
}

// parseProjectionCatchupTimeout reads PROJECTION_CATCHUP_TIMEOUT_S from env
// (positive integer seconds); default 5 minutes. Invalid values silently
// keep the default, matching the four service copies this replaces.
func parseProjectionCatchupTimeout() time.Duration {
	const def = 5 * time.Minute
	v := os.Getenv("PROJECTION_CATCHUP_TIMEOUT_S")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return time.Duration(n) * time.Second
}
