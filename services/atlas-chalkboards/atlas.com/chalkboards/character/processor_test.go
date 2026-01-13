package character

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func newTestContext(t tenant.Model) context.Context {
	return tenant.WithContext(context.Background(), t)
}

func newTestField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
}

func resetProcessorRegistry() {
	r := getRegistry()
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.characterRegister = make(map[MapKey][]uint32)
}

func TestProcessorInMapProvider(t *testing.T) {
	resetProcessorRegistry()
	ten := newTestTenant()
	ctx := newTestContext(ten)
	f := newTestField()
	l := logrus.New()

	p := NewProcessor(l, ctx)

	// Add a character first
	p.Enter(f, 12345)

	result, err := p.InMapProvider(f)()
	if err != nil {
		t.Fatalf("InMapProvider failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("Expected 1 character, got %d", len(result))
	}
	if result[0] != 12345 {
		t.Errorf("Expected character 12345, got %d", result[0])
	}
}

func TestProcessorGetCharactersInMap(t *testing.T) {
	resetProcessorRegistry()
	ten := newTestTenant()
	ctx := newTestContext(ten)
	f := newTestField()
	l := logrus.New()

	p := NewProcessor(l, ctx)

	p.Enter(f, 1)
	p.Enter(f, 2)

	result, err := p.GetCharactersInMap(f)
	if err != nil {
		t.Fatalf("GetCharactersInMap failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 characters, got %d", len(result))
	}
}

func TestProcessorEnter(t *testing.T) {
	resetProcessorRegistry()
	ten := newTestTenant()
	ctx := newTestContext(ten)
	f := newTestField()
	l := logrus.New()

	p := NewProcessor(l, ctx)

	p.Enter(f, 12345)

	result, _ := p.GetCharactersInMap(f)
	if len(result) != 1 || result[0] != 12345 {
		t.Errorf("Expected [12345], got %v", result)
	}
}

func TestProcessorExit(t *testing.T) {
	resetProcessorRegistry()
	ten := newTestTenant()
	ctx := newTestContext(ten)
	f := newTestField()
	l := logrus.New()

	p := NewProcessor(l, ctx)

	p.Enter(f, 12345)
	p.Exit(f, 12345)

	result, _ := p.GetCharactersInMap(f)
	if len(result) != 0 {
		t.Errorf("Expected empty, got %v", result)
	}
}

func TestProcessorTransitionMap(t *testing.T) {
	resetProcessorRegistry()
	ten := newTestTenant()
	ctx := newTestContext(ten)
	l := logrus.New()

	oldField := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	newField := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(200000000)).Build()

	p := NewProcessor(l, ctx)

	p.Enter(oldField, 12345)
	p.TransitionMap(oldField, newField, 12345)

	oldResult, _ := p.GetCharactersInMap(oldField)
	if len(oldResult) != 0 {
		t.Errorf("Expected empty old map, got %v", oldResult)
	}

	newResult, _ := p.GetCharactersInMap(newField)
	if len(newResult) != 1 || newResult[0] != 12345 {
		t.Errorf("Expected [12345] in new map, got %v", newResult)
	}
}

func TestProcessorTransitionChannel(t *testing.T) {
	resetProcessorRegistry()
	ten := newTestTenant()
	ctx := newTestContext(ten)
	l := logrus.New()

	oldField := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	newField := field.NewBuilder(world.Id(0), channel.Id(2), _map.Id(100000000)).Build()

	p := NewProcessor(l, ctx)

	p.Enter(oldField, 12345)
	p.TransitionChannel(oldField, newField, 12345)

	oldResult, _ := p.GetCharactersInMap(oldField)
	if len(oldResult) != 0 {
		t.Errorf("Expected empty old channel, got %v", oldResult)
	}

	newResult, _ := p.GetCharactersInMap(newField)
	if len(newResult) != 1 || newResult[0] != 12345 {
		t.Errorf("Expected [12345] in new channel, got %v", newResult)
	}
}

func TestProcessorTenantIsolation(t *testing.T) {
	resetProcessorRegistry()
	tenant1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tenant2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)
	ctx1 := newTestContext(tenant1)
	ctx2 := newTestContext(tenant2)
	f := newTestField()
	l := logrus.New()

	p1 := NewProcessor(l, ctx1)
	p2 := NewProcessor(l, ctx2)

	p1.Enter(f, 1)
	p2.Enter(f, 2)

	result1, _ := p1.GetCharactersInMap(f)
	result2, _ := p2.GetCharactersInMap(f)

	if len(result1) != 1 || result1[0] != 1 {
		t.Errorf("Tenant1 expected [1], got %v", result1)
	}
	if len(result2) != 1 || result2[0] != 2 {
		t.Errorf("Tenant2 expected [2], got %v", result2)
	}
}

func TestProcessorEmptyMap(t *testing.T) {
	resetProcessorRegistry()
	ten := newTestTenant()
	ctx := newTestContext(ten)
	f := newTestField()
	l := logrus.New()

	p := NewProcessor(l, ctx)

	result, err := p.GetCharactersInMap(f)
	if err != nil {
		t.Fatalf("GetCharactersInMap failed: %v", err)
	}
	// Note: GetInMap returns nil for non-existent keys, which is valid Go behavior
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %v", result)
	}
}
