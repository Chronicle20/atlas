package _map

import (
	"atlas-query-aggregator/map/mock"
	"errors"
	"testing"
)

func TestProcessorMock_GetPlayerCountInMap_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetPlayerCountInMapFunc: func(worldId byte, channelId byte, mapId uint32) (int, error) {
			if mapId == 100000000 {
				return 50, nil
			}
			return 0, nil
		},
	}

	count, err := mockProcessor.GetPlayerCountInMap(0, 0, 100000000)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if count != 50 {
		t.Errorf("Expected count=50, got %d", count)
	}
}

func TestProcessorMock_GetPlayerCountInMap_EmptyMap(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetPlayerCountInMapFunc: func(worldId byte, channelId byte, mapId uint32) (int, error) {
			return 0, nil
		},
	}

	count, err := mockProcessor.GetPlayerCountInMap(0, 0, 999999)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count=0, got %d", count)
	}
}

func TestProcessorMock_GetPlayerCountInMap_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetPlayerCountInMapFunc: func(worldId byte, channelId byte, mapId uint32) (int, error) {
			return 0, errors.New("map service unavailable")
		},
	}

	_, err := mockProcessor.GetPlayerCountInMap(0, 0, 100000000)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "map service unavailable" {
		t.Errorf("Expected error message 'map service unavailable', got '%s'", err.Error())
	}
}

func TestProcessorMock_GetPlayerCountInMap_DifferentChannels(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetPlayerCountInMapFunc: func(worldId byte, channelId byte, mapId uint32) (int, error) {
			// Different player counts per channel
			return int(channelId+1) * 10, nil
		},
	}

	// Test channel 0
	count, err := mockProcessor.GetPlayerCountInMap(0, 0, 100000000)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if count != 10 {
		t.Errorf("Expected count=10 for channel 0, got %d", count)
	}

	// Test channel 1
	count, err = mockProcessor.GetPlayerCountInMap(0, 1, 100000000)
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
	count, err := mockProcessor.GetPlayerCountInMap(0, 0, 100000000)
	if err != nil {
		t.Errorf("Expected no error from default GetPlayerCountInMap, got %v", err)
	}

	if count != 0 {
		t.Errorf("Expected default count=0, got %d", count)
	}
}
