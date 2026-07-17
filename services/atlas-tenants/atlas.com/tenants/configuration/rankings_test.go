package configuration_test

import (
	"atlas-tenants/configuration"
	"atlas-tenants/kafka/message"
	"atlas-tenants/test"
	"errors"
	"testing"

	"github.com/google/uuid"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func rankingsAttrs(minutes float64) map[string]interface{} {
	return map[string]interface{}{
		"type": "rankings",
		"id":   uuid.New().String(),
		"attributes": map[string]interface{}{
			"recomputeIntervalMinutes": minutes,
		},
	}
}

func setupRankingsProcessor(t *testing.T) (configuration.Processor, func()) {
	t.Helper()
	db := test.SetupTestDB(t)
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	cleanup := func() {
		test.CleanupTestDB(db)
	}
	return configuration.NewProcessor(logger, test.CreateTestContext(), db), cleanup
}

func TestRankingsCreateGetRoundTrip(t *testing.T) {
	p, cleanup := setupRankingsProcessor(t)
	defer cleanup()
	tenantId := uuid.New()
	mb := message.NewBuffer()

	if _, err := p.CreateRankings(mb)(tenantId)(rankingsAttrs(15)); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := p.GetRankings(tenantId)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	attrs, _ := got["attributes"].(map[string]interface{})
	if v, _ := attrs["recomputeIntervalMinutes"].(float64); v != 15 {
		t.Fatalf("interval = %v, want 15", attrs["recomputeIntervalMinutes"])
	}
}

func TestRankingsGetAbsentIsNotFound(t *testing.T) {
	p, cleanup := setupRankingsProcessor(t)
	defer cleanup()

	_, err := p.GetRankings(uuid.New())
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestRankingsUpdateReplaces(t *testing.T) {
	p, cleanup := setupRankingsProcessor(t)
	defer cleanup()
	tenantId := uuid.New()
	mb := message.NewBuffer()

	if _, err := p.CreateRankings(mb)(tenantId)(rankingsAttrs(15)); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := p.UpdateRankings(mb)(tenantId)(rankingsAttrs(45)); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := p.GetRankings(tenantId)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	attrs, _ := got["attributes"].(map[string]interface{})
	if v, _ := attrs["recomputeIntervalMinutes"].(float64); v != 45 {
		t.Fatalf("interval = %v, want 45", attrs["recomputeIntervalMinutes"])
	}
}

func TestRankingsUpdateAbsentFails(t *testing.T) {
	p, cleanup := setupRankingsProcessor(t)
	defer cleanup()

	_, err := p.UpdateRankings(message.NewBuffer())(uuid.New())(rankingsAttrs(45))
	if err == nil {
		t.Fatal("update of absent config must fail")
	}
}

func TestRankingsDelete(t *testing.T) {
	p, cleanup := setupRankingsProcessor(t)
	defer cleanup()
	tenantId := uuid.New()
	mb := message.NewBuffer()

	if _, err := p.CreateRankings(mb)(tenantId)(rankingsAttrs(15)); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := p.DeleteRankings(mb)(tenantId); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := p.GetRankings(tenantId); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not-found after delete, got %v", err)
	}
}

func TestRankingsTenantsIsolated(t *testing.T) {
	p, cleanup := setupRankingsProcessor(t)
	defer cleanup()
	tenantA := uuid.New()
	tenantB := uuid.New()
	mb := message.NewBuffer()

	if _, err := p.CreateRankings(mb)(tenantA)(rankingsAttrs(15)); err != nil {
		t.Fatalf("create A: %v", err)
	}
	if _, err := p.CreateRankings(mb)(tenantB)(rankingsAttrs(30)); err != nil {
		t.Fatalf("create B: %v", err)
	}

	gotA, err := p.GetRankings(tenantA)
	if err != nil {
		t.Fatalf("get A: %v", err)
	}
	attrsA, _ := gotA["attributes"].(map[string]interface{})
	if v, _ := attrsA["recomputeIntervalMinutes"].(float64); v != 15 {
		t.Fatalf("tenant A interval = %v, want 15", v)
	}

	gotB, err := p.GetRankings(tenantB)
	if err != nil {
		t.Fatalf("get B: %v", err)
	}
	attrsB, _ := gotB["attributes"].(map[string]interface{})
	if v, _ := attrsB["recomputeIntervalMinutes"].(float64); v != 30 {
		t.Fatalf("tenant B interval = %v, want 30", v)
	}
}
