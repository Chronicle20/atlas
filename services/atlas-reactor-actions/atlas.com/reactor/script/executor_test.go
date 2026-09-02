package script

import (
	"context"
	"testing"

	reactorsaga "atlas-reactor-actions/saga"

	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
)

// fakeSagaProcessor is a test double for reactorsaga.Processor that records
// every saga it is asked to create, so tests can assert on the payload of
// the step(s) produced without hitting a real Kafka producer.
type fakeSagaProcessor struct {
	created []saga.Saga
	err     error
}

func (f *fakeSagaProcessor) Create(s saga.Saga) error {
	if f.err != nil {
		return f.err
	}
	f.created = append(f.created, s)
	return nil
}

var _ reactorsaga.Processor = (*fakeSagaProcessor)(nil)

func newTestReactorContext() ReactorContext {
	return ReactorContext{
		Field:          field.NewBuilder(1, 2, 100000000).Build(),
		ReactorId:      1,
		Classification: "test_reactor",
		ReactorName:    "test_reactor",
		X:              100,
		Y:              200,
	}
}

func newTestExecutor(fake *fakeSagaProcessor) *OperationExecutor {
	logger, _ := test.NewNullLogger()
	return &OperationExecutor{
		l:     logger,
		ctx:   context.Background(),
		sagaP: fake,
	}
}

func TestExecuteDropItems_MesoParams(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]string
		wantMeso       bool
		wantMesoChance uint32
		wantMesoMin    uint32
		wantMesoMax    uint32
		wantMinItems   uint32
		wantDropType   string
	}{
		{
			name: "all new params",
			params: map[string]string{
				"meso":       "true",
				"mesoChance": "2",
				"mesoMin":    "8",
				"mesoMax":    "15",
				"minItems":   "1",
			},
			wantMeso:       true,
			wantMesoChance: 2,
			wantMesoMin:    8,
			wantMesoMax:    15,
			wantMinItems:   1,
			wantDropType:   "drop",
		},
		{
			name:           "no params",
			params:         map[string]string{},
			wantMeso:       false,
			wantMesoChance: 1,
			wantMesoMin:    1,
			wantMesoMax:    1,
			wantMinItems:   0,
			wantDropType:   "drop",
		},
		{
			name: "legacy minMeso ignored",
			params: map[string]string{
				"meso":    "true",
				"minMeso": "2",
				"maxMeso": "8",
			},
			wantMeso:       true,
			wantMesoChance: 1,
			wantMesoMin:    1,
			wantMesoMax:    1,
			wantMinItems:   0,
			wantDropType:   "drop",
		},
		{
			name: "legacy mixed with new",
			params: map[string]string{
				"meso":    "true",
				"mesoMin": "8",
				"minMeso": "999",
			},
			wantMeso:       true,
			wantMesoChance: 1,
			wantMesoMin:    8,
			wantMesoMax:    1,
			wantMinItems:   0,
			wantDropType:   "drop",
		},
		{
			name: "unparseable value falls back to default",
			params: map[string]string{
				"meso":       "true",
				"mesoChance": "abc",
			},
			wantMeso:       true,
			wantMesoChance: 1,
			wantMesoMin:    1,
			wantMesoMax:    1,
			wantMinItems:   0,
			wantDropType:   "drop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, err := operation.NewBuilder().SetType("drop_items").SetParams(tt.params).Build()
			if err != nil {
				t.Fatalf("failed to build operation: %v", err)
			}

			fake := &fakeSagaProcessor{}
			executor := newTestExecutor(fake)
			rc := newTestReactorContext()

			if err := executor.ExecuteOperation(rc, 12345, op); err != nil {
				t.Fatalf("ExecuteOperation() error = %v", err)
			}

			if len(fake.created) != 1 {
				t.Fatalf("expected 1 saga created, got %d", len(fake.created))
			}

			s := fake.created[0]
			if len(s.Steps) != 1 {
				t.Fatalf("expected 1 step, got %d", len(s.Steps))
			}

			payload, ok := s.Steps[0].Payload.(saga.SpawnReactorDropsPayload)
			if !ok {
				t.Fatalf("step payload is not a SpawnReactorDropsPayload: %T", s.Steps[0].Payload)
			}

			if payload.Meso != tt.wantMeso {
				t.Errorf("Meso = %v, want %v", payload.Meso, tt.wantMeso)
			}
			if payload.MesoChance != tt.wantMesoChance {
				t.Errorf("MesoChance = %d, want %d", payload.MesoChance, tt.wantMesoChance)
			}
			if payload.MesoMin != tt.wantMesoMin {
				t.Errorf("MesoMin = %d, want %d", payload.MesoMin, tt.wantMesoMin)
			}
			if payload.MesoMax != tt.wantMesoMax {
				t.Errorf("MesoMax = %d, want %d", payload.MesoMax, tt.wantMesoMax)
			}
			if payload.MinItems != tt.wantMinItems {
				t.Errorf("MinItems = %d, want %d", payload.MinItems, tt.wantMinItems)
			}
			if payload.DropType != tt.wantDropType {
				t.Errorf("DropType = %q, want %q", payload.DropType, tt.wantDropType)
			}
		})
	}
}

func TestExecuteSprayItems_SetsDropType(t *testing.T) {
	params := map[string]string{
		"meso":       "true",
		"mesoChance": "1",
		"mesoMin":    "50",
		"mesoMax":    "100",
		"minItems":   "15",
	}

	op, err := operation.NewBuilder().SetType("spray_items").SetParams(params).Build()
	if err != nil {
		t.Fatalf("failed to build operation: %v", err)
	}

	fake := &fakeSagaProcessor{}
	executor := newTestExecutor(fake)
	rc := newTestReactorContext()

	if err := executor.ExecuteOperation(rc, 12345, op); err != nil {
		t.Fatalf("ExecuteOperation() error = %v", err)
	}

	if len(fake.created) != 1 {
		t.Fatalf("expected 1 saga created, got %d", len(fake.created))
	}

	s := fake.created[0]
	if len(s.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(s.Steps))
	}

	payload, ok := s.Steps[0].Payload.(saga.SpawnReactorDropsPayload)
	if !ok {
		t.Fatalf("step payload is not a SpawnReactorDropsPayload: %T", s.Steps[0].Payload)
	}

	if payload.DropType != "spray" {
		t.Errorf("DropType = %q, want %q", payload.DropType, "spray")
	}
	if payload.MesoMin != 50 {
		t.Errorf("MesoMin = %d, want 50", payload.MesoMin)
	}
	if payload.MesoMax != 100 {
		t.Errorf("MesoMax = %d, want 100", payload.MesoMax)
	}
	if payload.MinItems != 15 {
		t.Errorf("MinItems = %d, want 15", payload.MinItems)
	}
}

// TestExecuteSprayItems_NoParams pins the invariant that executeSprayItems
// injects params["dropType"] = "spray" into whatever map op.Params() returns.
// operation.NewBuilder always seeds a non-nil params map, and
// convertJsonOperation only calls SetParams when the seed JSON has a
// non-nil params object, so a spray_items operation decoded from a seed
// with no params key must still execute without panicking.
func TestExecuteSprayItems_NoParams(t *testing.T) {
	op, err := operation.NewBuilder().SetType("spray_items").Build()
	if err != nil {
		t.Fatalf("failed to build operation: %v", err)
	}

	fake := &fakeSagaProcessor{}
	executor := newTestExecutor(fake)
	rc := newTestReactorContext()

	if err := executor.ExecuteOperation(rc, 12345, op); err != nil {
		t.Fatalf("ExecuteOperation() error = %v", err)
	}

	if len(fake.created) != 1 {
		t.Fatalf("expected 1 saga created, got %d", len(fake.created))
	}

	payload, ok := fake.created[0].Steps[0].Payload.(saga.SpawnReactorDropsPayload)
	if !ok {
		t.Fatalf("step payload is not a SpawnReactorDropsPayload: %T", fake.created[0].Steps[0].Payload)
	}
	if payload.DropType != "spray" {
		t.Errorf("DropType = %q, want %q", payload.DropType, "spray")
	}
}
