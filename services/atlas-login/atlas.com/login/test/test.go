package test

import (
	"context"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// TestLogger creates a null logger for testing
func TestLogger() logrus.FieldLogger {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	return logger
}

// TestTenant creates a test tenant model
func TestTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

// TestContext creates a context with a test tenant
func TestContext() context.Context {
	return tenant.WithContext(context.Background(), TestTenant())
}
