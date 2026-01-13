package drop

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus/hooks/test"
)

func TestNewExpirationTask_CreatesTaskWithCorrectValues(t *testing.T) {
	logger, _ := test.NewNullLogger()
	interval := 5 * time.Second

	task := NewExpirationTask(logger, interval)

	if task == nil {
		t.Fatal("Expected task to be created")
	}
	if task.l == nil {
		t.Fatal("Expected logger to be set")
	}
	if task.interval != interval {
		t.Fatalf("Expected interval %v, got %v", interval, task.interval)
	}
}

func TestExpirationTask_SleepTime_ReturnsInterval(t *testing.T) {
	logger, _ := test.NewNullLogger()
	interval := 10 * time.Second

	task := NewExpirationTask(logger, interval)

	if task.SleepTime() != interval {
		t.Fatalf("Expected SleepTime %v, got %v", interval, task.SleepTime())
	}
}

func TestExpirationTask_SleepTime_DifferentIntervals(t *testing.T) {
	logger, _ := test.NewNullLogger()

	tests := []time.Duration{
		1 * time.Second,
		30 * time.Second,
		1 * time.Minute,
		5 * time.Minute,
	}

	for _, interval := range tests {
		task := NewExpirationTask(logger, interval)
		if task.SleepTime() != interval {
			t.Errorf("Expected SleepTime %v, got %v", interval, task.SleepTime())
		}
	}
}

func TestExpirationTaskName_Constant(t *testing.T) {
	if ExpirationTaskName != "drop_expiration_task" {
		t.Fatalf("Expected ExpirationTaskName 'drop_expiration_task', got '%s'", ExpirationTaskName)
	}
}
