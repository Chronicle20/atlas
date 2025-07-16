package tracing

import (
	"io"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func TestInitTracer(t *testing.T) {
	logger := logrus.New()
	serviceName := "test-service"
	
	tracerFunc := InitTracer(logger)
	
	if tracerFunc == nil {
		t.Error("InitTracer() returned nil")
	}
	
	closer, err := tracerFunc(serviceName)
	
	if err != nil {
		t.Errorf("InitTracer() returned error: %v", err)
	}
	
	if closer == nil {
		t.Error("InitTracer() returned nil closer")
	}
}

func TestInitTracer_DifferentServices(t *testing.T) {
	logger := logrus.New()
	
	tracerFunc := InitTracer(logger)
	
	// Test with different service names
	service1 := "service1"
	service2 := "service2"
	
	closer1, err1 := tracerFunc(service1)
	closer2, err2 := tracerFunc(service2)
	
	if err1 != nil {
		t.Errorf("InitTracer() returned error for service1: %v", err1)
	}
	
	if err2 != nil {
		t.Errorf("InitTracer() returned error for service2: %v", err2)
	}
	
	if closer1 == nil {
		t.Error("InitTracer() returned nil closer for service1")
	}
	
	if closer2 == nil {
		t.Error("InitTracer() returned nil closer for service2")
	}
}

func TestTeardown(t *testing.T) {
	logger := logrus.New()
	
	teardownFunc := Teardown(logger)
	
	if teardownFunc == nil {
		t.Error("Teardown() returned nil")
	}
	
	// Create a mock closer
	var mockCloser io.Closer = &mockCloser{}
	
	teardownFuncWithProvider := teardownFunc(mockCloser)
	if teardownFuncWithProvider == nil {
		t.Error("Teardown()(closer) returned nil")
	}
	
	// Test that we can call the teardown function
	teardownFuncWithProvider()
}

func TestStartSpan(t *testing.T) {
	logger := logrus.New()
	
	spanName := "test-span"
	spanLogger, span := StartSpan(logger, spanName)
	
	if spanLogger == nil {
		t.Error("StartSpan() returned nil logger")
	}
	
	if span == nil {
		t.Error("StartSpan() returned nil span")
	}
	
	// Test that the span logger is different from the original
	if spanLogger == logger {
		t.Error("StartSpan() returned the same logger instance")
	}
}

func TestStartSpan_WithOptions(t *testing.T) {
	logger := logrus.New()
	
	spanName := "test-span"
	opts := []opentracing.StartSpanOption{
		opentracing.Tag{Key: "test", Value: "value"},
	}
	
	spanLogger, span := StartSpan(logger, spanName, opts...)
	
	if spanLogger == nil {
		t.Error("StartSpan() with options returned nil logger")
	}
	
	if span == nil {
		t.Error("StartSpan() with options returned nil span")
	}
}

func TestLogrusAdapter_Error(t *testing.T) {
	logger := logrus.New()
	adapter := LogrusAdapter{logger: logger}
	
	// Test that Error method exists and can be called
	adapter.Error("test error message")
	
	// No assertion needed, just verify it doesn't panic
}

func TestLogrusAdapter_Infof(t *testing.T) {
	logger := logrus.New()
	adapter := LogrusAdapter{logger: logger}
	
	// Test that Infof method exists and can be called
	adapter.Infof("test info message: %s", "arg1")
	
	// No assertion needed, just verify it doesn't panic
}

func TestLogrusAdapter_Structure(t *testing.T) {
	logger := logrus.New()
	adapter := LogrusAdapter{logger: logger}
	
	if adapter.logger != logger {
		t.Error("LogrusAdapter did not store the logger correctly")
	}
}

// mockCloser implements io.Closer for testing
type mockCloser struct {
	closed bool
}

func (m *mockCloser) Close() error {
	m.closed = true
	return nil
}

func TestTeardown_WithError(t *testing.T) {
	logger := logrus.New()
	
	teardownFunc := Teardown(logger)
	
	// Create a mock closer that returns an error
	var mockCloser io.Closer = &errorCloser{}
	
	teardownFuncWithProvider := teardownFunc(mockCloser)
	if teardownFuncWithProvider == nil {
		t.Error("Teardown()(closer) returned nil")
	}
	
	// Test that we can call the teardown function even with an error
	teardownFuncWithProvider()
}

// errorCloser implements io.Closer and returns an error
type errorCloser struct{}

func (e *errorCloser) Close() error {
	return &testError{message: "close error"}
}

// testError implements error interface
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}