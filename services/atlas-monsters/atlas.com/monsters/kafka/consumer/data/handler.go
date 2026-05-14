// Package data subscribes to EVENT_TOPIC_DATA for tenant-scoped cache
// invalidation events. We use a shared (single-delivery) consumer group
// "Monster Data Cache Invalidator" rather than a per-pod group because the
// cache state is in shared Redis (task-060 v2): one pod's Clear is visible
// to every replica immediately. If a future cache moves in-process, this
// consumer will need per-pod fan-out — see
// docs/tasks/task-061-data-cache-invalidation/design.md §6.3.
package data

import (
	"context"
	"os"
	"strconv"

	"atlas-monsters/monster/information"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func handleDataUpdated(l logrus.FieldLogger, ctx context.Context, e event[dataUpdatedEventBody]) {
	if e.Type != EventTypeDataUpdated {
		eventsSkippedTotal.WithLabelValues("unknown_type").Inc()
		return
	}
	if e.Body.Worker != WorkerMonster {
		eventsSkippedTotal.WithLabelValues("unrelated_worker").Inc()
		return
	}

	bodyTid, err := uuid.Parse(e.Body.TenantId)
	if err != nil {
		l.WithError(err).Errorf("DATA_UPDATED with malformed tenantId [%s]; ignoring.", e.Body.TenantId)
		eventsConsumerErrorsTotal.WithLabelValues("parse").Inc()
		return
	}

	headerTenant, terr := tenant.FromContext(ctx)()
	hasHeader := terr == nil
	var t tenant.Model
	if hasHeader && headerTenant.Id() == bodyTid {
		t = headerTenant
	} else {
		if hasHeader {
			l.Warnf("Tenant header [%s] disagrees with event body tenant [%s]; using body.", headerTenant.Id(), bodyTid)
		}
		fb, ferr := tenant.Create(bodyTid, "", 0, 0)
		if ferr != nil {
			l.WithError(ferr).Errorf("Failed to construct fallback tenant for [%s].", bodyTid)
			eventsConsumerErrorsTotal.WithLabelValues("parse").Inc()
			return
		}
		t = fb
	}

	deleted, ferr := information.FlushTenant(ctx, t)
	if ferr != nil {
		l.WithError(ferr).Errorf("Monster data cache flush partially failed for tenant [%s] (deleted [%d] keys before error).", bodyTid, deleted)
		eventsConsumerErrorsTotal.WithLabelValues("flush").Inc()
	}
	keysDeletedTotal.WithLabelValues(bodyTid.String()).Add(float64(deleted))
	eventsProcessedTotal.WithLabelValues(WorkerMonster, EventTypeDataUpdated, "flushed").Inc()
	l.Debugf("Flushed [%d] monster data cache keys for tenant [%s] in response to DATA_UPDATED.", deleted, bodyTid)
}

func consumerEnabled() bool {
	v, ok := os.LookupEnv("DATA_EVENTS_CONSUMER_ENABLED")
	if !ok {
		return true
	}
	enabled, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return enabled
}
