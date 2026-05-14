package monsterbook

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSetCoverEncodeShape(t *testing.T) {
	body := SetCover{CardId: 2380000}
	out := body.Encode(logrus.New(), context.Background())(map[string]interface{}{})
	if len(out) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(out))
	}
}
