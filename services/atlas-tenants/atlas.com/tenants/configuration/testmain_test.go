package configuration_test

import (
	"atlas-tenants/configuration"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(configuration.EventTopicConfigurationStatus), string(configuration.EventTopicConfigurationStatus))
	os.Exit(m.Run())
}
