package playernpc

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func TestCodeFor(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"pool exhausted sentinel", ErrPoolExhausted, CodePoolExhausted},
		{"wrapped pool exhausted", fmt.Errorf("deploy: %w", ErrPoolExhausted), CodePoolExhausted},
		{"map full sentinel", ErrMapFull, CodeMapFull},
		{"wrapped map full", fmt.Errorf("deploy: %w", ErrMapFull), CodeMapFull},
		{"duplicate sentinel", ErrDuplicate, CodeDuplicate},
		{"ineligible sentinel", ErrIneligible, CodeIneligible},
		{"not found", requests.ErrNotFound, CodeUnresolvable},
		{"wrapped not found", fmt.Errorf("character: %w", requests.ErrNotFound), CodeUnresolvable},
		{"unclassified error", errors.New("boom"), CodeInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CodeFor(tt.err); got != tt.want {
				t.Errorf("CodeFor(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}
