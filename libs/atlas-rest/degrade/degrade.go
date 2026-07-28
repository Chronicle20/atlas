// Package degrade is the loud-degradation observer: every enrichment or
// fallback path that drops data on failure must call Observe so the
// degradation is logged and counted — degraded results must never be
// indistinguishable from correct ones.
package degrade

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var degradedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "atlas_enrichment_degraded_total",
		Help: "Number of enrichment/decorator failures that degraded to a partial result, by component.",
	},
	[]string{"component"},
)

// Observe logs the degradation at Warn with the entity id and cause, and
// increments atlas_enrichment_degraded_total{component}. component must be a
// static low-cardinality string (e.g. "login.character.inventory"); entityId
// goes only into the log line, never into a metric label.
func Observe(l logrus.FieldLogger, component string, entityId uint32, err error) {
	degradedTotal.WithLabelValues(component).Inc()
	l.WithError(err).Warnf("Enrichment degraded for component [%s], entity [%d]; returning un-enriched model.", component, entityId)
}
