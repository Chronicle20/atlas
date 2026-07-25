package degrade

import (
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestObserveLogsWarnAndCounts(t *testing.T) {
	logger, hook := test.NewNullLogger()
	before := testutil.ToFloat64(degradedTotal.WithLabelValues("test.component"))

	Observe(logger, "test.component", 42, errors.New("fetch failed"))

	after := testutil.ToFloat64(degradedTotal.WithLabelValues("test.component"))
	if after-before != 1 {
		t.Fatalf("expected counter delta 1, got %v", after-before)
	}
	entry := hook.LastEntry()
	if entry == nil || entry.Level != logrus.WarnLevel {
		t.Fatalf("expected a Warn entry, got %+v", entry)
	}
	if !strings.Contains(entry.Message, "test.component") || !strings.Contains(entry.Message, "42") {
		t.Fatalf("Warn must name component and entity id, got: %s", entry.Message)
	}
}
