package tenant_test

import (
	"atlas-tenants/tenant"
	"regexp"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

// envVarNamePattern is the shape a topic.Token must have. A topic.Token is
// the NAME OF AN ENVIRONMENT VARIABLE (libs/atlas-kafka/topic/token.go), and
// every topic env var the overlays declare is SCREAMING_SNAKE_CASE.
var envVarNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// TestEventTopicTenantStatusIsATokenNotATopicName pins the task-288
// regression. The constant was previously the untyped string "tenant.status"
// -- a Kafka topic NAME, not an environment variable name -- which reached
// Buffer.Put's topic.Token parameter by implicit conversion and was then
// handed to topic.EnvProvider as a variable to look up. No overlay declares
// such a variable, so once EnvProvider stopped falling back to the token
// string every tenant emit failed with HTTP 500.
//
// The legacy value is carried as a negative case so the assertion is shown to
// discriminate: a check that passes for both shapes would not have caught the
// bug. Resolution behaviour itself belongs to EnvProvider and is already
// covered by libs/atlas-kafka/topic/provider_test.go; duplicating it here
// would not fail against the pre-fix constant.
func TestEventTopicTenantStatusIsATokenNotATopicName(t *testing.T) {
	tests := []struct {
		name           string
		token          topic.Token
		wantEnvVarName bool
	}{
		{
			name:           "declared token",
			token:          tenant.EventTopicTenantStatus,
			wantEnvVarName: true,
		},
		{
			name:           "legacy topic name the service regressed on",
			token:          "tenant.status",
			wantEnvVarName: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := envVarNamePattern.MatchString(string(tt.token))
			if got != tt.wantEnvVarName {
				t.Fatalf("%q is an environment variable name = %v, want %v; a topic.Token names the env var carrying the topic, never the topic itself", tt.token, got, tt.wantEnvVarName)
			}
			if tt.wantEnvVarName && !strings.HasPrefix(string(tt.token), "EVENT_TOPIC_") {
				t.Errorf("%q lacks the EVENT_TOPIC_ prefix the fleet-wide token convention requires", tt.token)
			}
		})
	}
}
