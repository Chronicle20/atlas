package characterrender

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	renderTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "character_render_total",
		Help: "Total number of character renders served, labelled by stance and two-handed override.",
	}, []string{"stance", "two_handed_override"})

	renderErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "character_render_errors_total",
		Help: "Total number of character render errors, labelled by reason code.",
	}, []string{"reason"})

	renderDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "character_render_duration_ms",
		Help:    "Render duration in milliseconds (cache miss path only).",
		Buckets: []float64{50, 100, 200, 300, 500, 750, 1000, 1500, 2000, 3000},
	})
)

// IncrementRender records a successful render outcome.
func IncrementRender(stance string, override bool) {
	tag := "false"
	if override {
		tag = "true"
	}
	renderTotal.WithLabelValues(stance, tag).Inc()
}

// IncrementError records a failed render outcome.
func IncrementError(reason string) { renderErrors.WithLabelValues(reason).Inc() }

// ObserveDurationMs records a render duration sample.
func ObserveDurationMs(ms float64) { renderDuration.Observe(ms) }
