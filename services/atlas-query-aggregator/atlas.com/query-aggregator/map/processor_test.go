package _map

import (
	"atlas-query-aggregator/map/mock"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

func TestProcessorMock_GetPlayerCountInMap_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetPlayerCountInMapFunc: func(f field.Model) (int, error) {
			if f.MapId() == 100000000 {
				return 50, nil
			}
			return 0, nil
		},
	}

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	count, err := mockProcessor.GetPlayerCountInMap(f)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if count != 50 {
		t.Errorf("Expected count=50, got %d", count)
	}
}

func TestProcessorMock_GetPlayerCountInMap_EmptyMap(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetPlayerCountInMapFunc: func(f field.Model) (int, error) {
			return 0, nil
		},
	}

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(999999)).SetInstance(uuid.Nil).Build()
	count, err := mockProcessor.GetPlayerCountInMap(f)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count=0, got %d", count)
	}
}

func TestProcessorMock_GetPlayerCountInMap_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetPlayerCountInMapFunc: func(f field.Model) (int, error) {
			return 0, errors.New("map service unavailable")
		},
	}

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	_, err := mockProcessor.GetPlayerCountInMap(f)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "map service unavailable" {
		t.Errorf("Expected error message 'map service unavailable', got '%s'", err.Error())
	}
}

func TestProcessorMock_GetPlayerCountInMap_DifferentChannels(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetPlayerCountInMapFunc: func(f field.Model) (int, error) {
			// Different player counts per channel
			return int(f.ChannelId()+1) * 10, nil
		},
	}

	// Test channel 0
	f0 := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	count, err := mockProcessor.GetPlayerCountInMap(f0)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if count != 10 {
		t.Errorf("Expected count=10 for channel 0, got %d", count)
	}

	// Test channel 1
	f1 := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	count, err = mockProcessor.GetPlayerCountInMap(f1)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if count != 20 {
		t.Errorf("Expected count=20 for channel 1, got %d", count)
	}
}

func TestProcessorMock_DefaultBehavior(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{}

	// Test default GetPlayerCountInMap returns 0
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	count, err := mockProcessor.GetPlayerCountInMap(f)
	if err != nil {
		t.Errorf("Expected no error from default GetPlayerCountInMap, got %v", err)
	}

	if count != 0 {
		t.Errorf("Expected default count=0, got %d", count)
	}
}
