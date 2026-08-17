package producer

import (
	"context"
	"testing"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

func TestEnvHeaderDecoratorEmitsTheEnvironment(t *testing.T) {
	ctx := env.WithContext(context.Background(), env.Id("pr-123"))
	got, err := EnvHeaderDecorator(ctx)()
	if err != nil {
		t.Fatalf("decorator: %v", err)
	}
	if got[env.Key] != "pr-123" {
		t.Fatalf("headers = %v, want %s=pr-123", got, env.Key)
	}
}

func TestEnvHeaderDecoratorEmitsNothingForTheLegacyEnvironment(t *testing.T) {
	// NFR-7: a main-only message is byte-identical to today's.
	got, err := EnvHeaderDecorator(context.Background())()
	if err != nil {
		t.Fatalf("decorator: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("headers = %v, want none", got)
	}
}
