package logger

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"go.elastic.co/ecslogrus"
)

func TestCreateLogger(t *testing.T) {
	serviceName := "test-service"
	logger := CreateLogger(serviceName)
	
	if logger == nil {
		t.Error("CreateLogger() returned nil")
	}
	
	// Test that the formatter is set to ECS
	if _, ok := logger.Formatter.(*ecslogrus.Formatter); !ok {
		t.Error("Logger formatter is not ECS formatter")
	}
	
	// Test that hooks are added
	if len(logger.Hooks) == 0 {
		t.Error("Logger has no hooks")
	}
	
	// Test that the service name hook is present
	found := false
	for _, hooks := range logger.Hooks {
		for _, hook := range hooks {
			if efh, ok := hook.(*ExtraFieldHook); ok {
				if efh.service == serviceName {
					found = true
					break
				}
			}
		}
	}
	
	if !found {
		t.Error("Service name hook not found or incorrect service name")
	}
}

func TestCreateLogger_WithLogLevel(t *testing.T) {
	// Set environment variable
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")
	
	logger := CreateLogger("test-service")
	
	if logger.Level != logrus.DebugLevel {
		t.Errorf("Expected log level to be DebugLevel, got %v", logger.Level)
	}
}

func TestCreateLogger_WithInvalidLogLevel(t *testing.T) {
	// Set invalid environment variable
	os.Setenv("LOG_LEVEL", "invalid")
	defer os.Unsetenv("LOG_LEVEL")
	
	logger := CreateLogger("test-service")
	
	// Should default to Info level
	if logger.Level != logrus.InfoLevel {
		t.Errorf("Expected log level to be InfoLevel, got %v", logger.Level)
	}
}

func TestNewHook(t *testing.T) {
	serviceName := "test-service"
	hook := newHook(serviceName)
	
	if hook == nil {
		t.Error("newHook() returned nil")
	}
	
	if hook.service != serviceName {
		t.Errorf("Expected service name '%s', got '%s'", serviceName, hook.service)
	}
}

func TestExtraFieldHook_Levels(t *testing.T) {
	hook := &ExtraFieldHook{service: "test"}
	
	levels := hook.Levels()
	
	if len(levels) != len(logrus.AllLevels) {
		t.Errorf("Expected %d levels, got %d", len(logrus.AllLevels), len(levels))
	}
	
	// Check that all levels are present
	for _, expectedLevel := range logrus.AllLevels {
		found := false
		for _, level := range levels {
			if level == expectedLevel {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Level %v not found in hook levels", expectedLevel)
		}
	}
}

func TestExtraFieldHook_Fire(t *testing.T) {
	serviceName := "test-service"
	hook := &ExtraFieldHook{service: serviceName}
	
	entry := &logrus.Entry{
		Data: make(logrus.Fields),
	}
	
	err := hook.Fire(entry)
	
	if err != nil {
		t.Errorf("Fire() returned error: %v", err)
	}
	
	if entry.Data["service.name"] != serviceName {
		t.Errorf("Expected service.name to be '%s', got '%v'", serviceName, entry.Data["service.name"])
	}
}