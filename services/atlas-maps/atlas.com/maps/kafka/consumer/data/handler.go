// Package data subscribes to EVENT_TOPIC_DATA for tenant-scoped spawn-
// registry invalidation. We use a shared (single-delivery) consumer group
// "Map Spawn Registry Invalidator" because the spawn registry lives in
// shared Redis: one pod's FlushTenant is visible to every replica
// immediately. See docs/tasks/task-061-data-cache-invalidation/design.md
// §6.3 for the rationale.
package data

import (
	"context"
	"os"
	"strconv"

	spawnMonster "atlas-maps/map/monster"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func handleDataUpdated(l logrus.FieldLogger, ctx context.Context, e event[dataUpdatedEventBody]) {
	if e.Type != EventTypeDataUpdated {
		eventsSkippedTotal.WithLabelValues("unknown_type").Inc()
		return
	}
	if e.Body.Worker != WorkerMap {
		eventsSkippedTotal.WithLabelValues("unrelated_worker").Inc()
		return
	}

	tid, err := uuid.Parse(e.Body.TenantId)
	if err != nil {
		l.WithError(err).Errorf("DATA_UPDATED with malformed tenantId [%s]; ignoring.", e.Body.TenantId)
		eventsConsumerErrorsTotal.WithLabelValues("parse").Inc()
		return
	}

	deleted, ferr := spawnMonster.GetRegistry().FlushTenant(ctx, l, tid)
	if ferr != nil {
		l.WithError(ferr).Errorf("Spawn-registry flush partially failed for tenant [%s] (deleted [%d] keys before error).", tid, deleted)
		eventsConsumerErrorsTotal.WithLabelValues("flush").Inc()
	}
	keysDeletedTotal.WithLabelValues(tid.String()).Add(float64(deleted))
	eventsProcessedTotal.WithLabelValues(WorkerMap, EventTypeDataUpdated, "flushed").Inc()
	l.Debugf("Flushed [%d] spawn-registry keys for tenant [%s] in response to DATA_UPDATED.", deleted, tid)
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
