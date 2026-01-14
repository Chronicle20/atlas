package route_test

import (
	"atlas-channel/transport/route"
	"testing"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	builder := route.NewModelBuilder("TestRoute")
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	id := uuid.New()
	model, err := route.NewModelBuilder("TestRoute").
		SetId(id).
		SetStartMapId(_map.Id(100000000)).
		SetStagingMapId(_map.Id(100000001)).
		SetDestinationMapId(_map.Id(100000002)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(1 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != id {
		t.Errorf("model.Id() = %v, want %v", model.Id(), id)
	}
	if model.Name() != "TestRoute" {
		t.Errorf("model.Name() = %s, want TestRoute", model.Name())
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	model := route.NewModelBuilder("TestRoute").MustBuild()
	if model.Name() != "TestRoute" {
		t.Errorf("model.Name() = %s, want TestRoute", model.Name())
	}
}

func TestCloneModel(t *testing.T) {
	original := route.NewModelBuilder("TestRoute").
		SetStartMapId(_map.Id(100000000)).
		MustBuild()

	cloned, err := route.CloneModel(original).
		SetStartMapId(_map.Id(200000000)).
		Build()

	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	if original.StartMapId() != _map.Id(100000000) {
		t.Errorf("original.StartMapId() = %d, want 100000000", original.StartMapId())
	}

	if cloned.StartMapId() != _map.Id(200000000) {
		t.Errorf("cloned.StartMapId() = %d, want 200000000", cloned.StartMapId())
	}
}

func TestTripScheduleBuilder(t *testing.T) {
	now := time.Now()
	model, err := route.NewTripScheduleBuilder().
		SetBoardingOpen(now).
		SetBoardingClosed(now.Add(5 * time.Minute)).
		SetDeparture(now.Add(6 * time.Minute)).
		SetArrival(now.Add(16 * time.Minute)).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.BoardingOpen() != now {
		t.Errorf("model.BoardingOpen() mismatch")
	}
}
