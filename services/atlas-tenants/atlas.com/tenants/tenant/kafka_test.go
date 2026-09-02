package tenant_test

import (
	"atlas-tenants/tenant"
	"os"
	"regexp"
	"strings"
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

// envVarNamePattern is the shape a topic.Token must have: a topic.Token is
// the NAME OF AN ENVIRONMENT VARIABLE (libs/atlas-kafka/topic/token.go), and
// the overlays declare every topic env var in SCREAMING_SNAKE_CASE.
var envVarNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// TestEventTopicTenantStatusIsAnEnvVarName pins the regression directly.
// The constant was previously the untyped string "tenant.status" -- a Kafka
// topic NAME, not an environment variable name -- which reached Buffer.Put's
// topic.Token parameter by implicit conversion and was then handed to
// topic.EnvProvider as a variable to look up. No such variable is declared in
// any overlay, so once EnvProvider stopped falling back to the token string
// every tenant emit failed with HTTP 500.
func TestEventTopicTenantStatusIsAnEnvVarName(t *testing.T) {
	got := string(tenant.EventTopicTenantStatus)

	if !envVarNamePattern.MatchString(got) {
		t.Fatalf("EventTopicTenantStatus = %q, which is not an environment variable name; a topic.Token names the env var carrying the topic, never the topic itself", got)
	}
	if !strings.HasPrefix(got, "EVENT_TOPIC_") {
		t.Errorf("EventTopicTenantStatus = %q, want an EVENT_TOPIC_ prefix to match the fleet-wide token convention", got)
	}
}

// TestEventTopicTenantStatusResolvesWhenSet covers the success half of the
// emit path's token resolution without standing up Kafka: EnvProvider is the
// exact component that returned the error behind the 500.
func TestEventTopicTenantStatusResolvesWhenSet(t *testing.T) {
	const resolved = "tenant-status-resolved"
	t.Setenv(string(tenant.EventTopicTenantStatus), resolved)

	logger, _ := logtest.NewNullLogger()

	got, err := topic.EnvProvider(logger)(tenant.EventTopicTenantStatus)()
	if err != nil {
		t.Fatalf("resolving %s: %v", tenant.EventTopicTenantStatus, err)
	}
	if got != resolved {
		t.Errorf("resolved topic = %q, want %q", got, resolved)
	}
}

// TestEventTopicTenantStatusErrorsWhenUnset pins the other half: an unset
// token must be a hard error, not a silent fallback to the token string.
// That fallback is what hid this bug until it was removed.
func TestEventTopicTenantStatusErrorsWhenUnset(t *testing.T) {
	// TestMain sets the token for the rest of the package; clear it here and
	// let t.Cleanup restore the process environment.
	prev, had := os.LookupEnv(string(tenant.EventTopicTenantStatus))
	if err := os.Unsetenv(string(tenant.EventTopicTenantStatus)); err != nil {
		t.Fatalf("unsetting %s: %v", tenant.EventTopicTenantStatus, err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(string(tenant.EventTopicTenantStatus), prev)
			return
		}
		_ = os.Unsetenv(string(tenant.EventTopicTenantStatus))
	})

	logger, _ := logtest.NewNullLogger()

	if _, err := topic.EnvProvider(logger)(tenant.EventTopicTenantStatus)(); err == nil {
		t.Fatal("resolving an unset token returned no error; an unset topic token must fail loudly")
	}
}
