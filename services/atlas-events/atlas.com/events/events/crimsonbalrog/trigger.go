package crimsonbalrog

import (
	"atlas-events/event/definition"
	"atlas-events/event/scheduling"
	transport "atlas-events/kafka/message/transport"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TriggerProcessor turns a VOYAGE_DEPARTED transport event into durable
// delayed work. It is a processor, not consumer code (FR-N18) — the
// consumer's job is only to decode the Kafka message, guard on type, and
// delegate here.
type TriggerProcessor interface {
	OnVoyageDeparted(e transport.StatusEvent[transport.VoyageStatusEventBody]) error
}

// TriggerProcessorImpl is the CRIMSON_BALROG TriggerProcessor.
//
// rollJitter is a field, not a package function, so a test can pin the
// rolled value rather than asserting only that it fell within a range. The
// jitter is rolled HERE, at scheduling time, and persisted in executeAt —
// rolling it at execution time would make the delay non-durable across a
// restart, which FR-B5 and the "the configured delay survives restart"
// acceptance criterion both forbid.
type TriggerProcessorImpl struct {
	l          logrus.FieldLogger
	ctx        context.Context
	db         *gorm.DB
	rollJitter func(time.Duration) time.Duration
}

// NewTriggerProcessor constructs a TriggerProcessor using the production
// jitter source: a uniform random duration in [0, d].
func NewTriggerProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *TriggerProcessorImpl {
	return NewTriggerProcessorWithJitter(l, ctx, db, defaultRollJitter)
}

// NewTriggerProcessorWithJitter constructs a TriggerProcessor with an
// injected jitter source, so a test can pin the rolled value.
func NewTriggerProcessorWithJitter(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, rollJitter func(time.Duration) time.Duration) *TriggerProcessorImpl {
	return &TriggerProcessorImpl{l: l, ctx: ctx, db: db, rollJitter: rollJitter}
}

var _ TriggerProcessor = (*TriggerProcessorImpl)(nil)

func defaultRollJitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(d) + 1))
}

// OnVoyageDeparted turns a departure into durable delayed work — one row per
// enabled, applicable definition (FR-B2), and NO occurrence (FR-B3): whether
// an attack happens is decided minutes later, when the delay elapses.
//
// Route matching is done locally: Config.ApplicableRouteIds holds route
// SLUGS, while the event's RouteId is a tenant-derived uuid
// (tenant.DerivedId(tenantId, "routes", slug) —
// events/crimsonbalrog/config.go). There is no REST call on this path.
//
// Deduplication (FR-B4: a redelivery must be a no-op) is enforced by
// scheduling.Administrator.Schedule via the dedupe key, not by a
// read-then-write race here.
func (p *TriggerProcessorImpl) OnVoyageDeparted(e transport.StatusEvent[transport.VoyageStatusEventBody]) error {
	t := tenant.MustFromContext(p.ctx)

	ds, err := definition.NewProcessor(p.l, p.ctx, p.db).GetEnabledByType(TypeName)
	if err != nil {
		return err
	}

	for _, d := range ds {
		c, err := DecodeConfig(d.Configuration())
		if err != nil {
			p.l.WithError(err).Warnf("Skipping definition [%s] with undecodable configuration.", d.Id())
			continue
		}

		matched := false
		for _, slug := range c.ApplicableRouteIds {
			if tenant.DerivedId(t.Id(), "routes", slug) == e.RouteId {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		wc := WorkContext{
			VoyageId:         e.Body.VoyageId,
			RouteId:          e.RouteId,
			WorldId:          e.Body.WorldId,
			ChannelId:        e.Body.ChannelId,
			StagingMapId:     e.Body.StagingMapId,
			EnRouteMapIds:    e.Body.EnRouteMapIds,
			DestinationMapId: e.Body.DestinationMapId,
			ObservationMapId: e.Body.ObservationMapId,
			DepartedAt:       e.Body.DepartedAt,
		}
		raw, err := json.Marshal(wc)
		if err != nil {
			return err
		}

		executeAt := e.Body.DepartedAt.Add(c.TriggerDelay.AsDuration()).Add(p.rollJitter(c.TriggerDelayJitter.AsDuration()))

		m, err := scheduling.NewBuilder(d.Id(), scheduling.WorkTypeTriggerEvaluation).
			SetExecuteAt(executeAt).
			SetContext(raw).
			SetDedupeKey(fmt.Sprintf("balrog:%s:%s:%d:%d", d.Id(), wc.VoyageId, wc.WorldId, wc.ChannelId)).
			Build()
		if err != nil {
			return err
		}

		if _, _, err := scheduling.NewAdministrator(p.l, p.ctx, p.db).Schedule(m); err != nil {
			return err
		}
	}

	return nil
}
