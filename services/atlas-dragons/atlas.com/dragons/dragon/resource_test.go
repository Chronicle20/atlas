package dragon

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestTransform(t *testing.T) {
	instance := uuid.New()
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(3)).SetInstance(instance).Build()
	m := NewBuilder(100).
		SetField(f).
		SetX(10).
		SetY(20).
		SetStance(5).
		SetJobId(job.Id(2200)).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.Id != "100" {
		t.Errorf("Id mismatch. Expected 100, got %v", rm.Id)
	}

	if rm.OwnerCharacterId != 100 {
		t.Errorf("OwnerCharacterId mismatch. Expected 100, got %v", rm.OwnerCharacterId)
	}

	if rm.X != 10 {
		t.Errorf("X mismatch. Expected 10, got %v", rm.X)
	}

	if rm.Y != 20 {
		t.Errorf("Y mismatch. Expected 20, got %v", rm.Y)
	}

	if rm.Stance != 5 {
		t.Errorf("Stance mismatch. Expected 5, got %v", rm.Stance)
	}

	if rm.JobId != uint16(job.Id(2200)) {
		t.Errorf("JobId mismatch. Expected %v, got %v", uint16(job.Id(2200)), rm.JobId)
	}

	if rm.WorldId != world.Id(1) {
		t.Errorf("WorldId mismatch. Expected %v, got %v", world.Id(1), rm.WorldId)
	}

	if rm.ChannelId != channel.Id(2) {
		t.Errorf("ChannelId mismatch. Expected %v, got %v", channel.Id(2), rm.ChannelId)
	}

	if rm.MapId != _map.Id(3) {
		t.Errorf("MapId mismatch. Expected %v, got %v", _map.Id(3), rm.MapId)
	}

	if rm.Instance != instance {
		t.Errorf("Instance mismatch. Expected %v, got %v", instance, rm.Instance)
	}
}
