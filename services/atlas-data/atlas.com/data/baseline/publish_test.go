package baseline

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestPublishErrorIsContextualized asserts Publisher.Publish wraps every
// failure path with a `publish: <step>:` prefix so operators can locate the
// failing step in logs. Pre-fix Publisher returned raw errors (or empty
// io.Pipe failure modes), producing the empty-500 observed on atlas-main.
func TestPublishErrorIsContextualized(t *testing.T) {
	p := Publisher{DB: nil, MC: nil, L: logrus.New()}
	_, err := p.Publish(context.Background(), "GMS", 83, 1)
	if err == nil {
		t.Fatal("expected error from Publish with nil deps")
	}
	if !strings.Contains(err.Error(), "publish:") {
		t.Fatalf("error %q lacks `publish:` step prefix", err.Error())
	}
}
