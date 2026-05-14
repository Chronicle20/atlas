package monsterbook

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSetCardEncodeShape(t *testing.T) {
	body := SetCard{CardId: 2380000, Level: 3, Added: true}
	out := body.Encode(logrus.New(), context.Background())(map[string]interface{}{})
	if len(out) != 9 {
		t.Fatalf("expected 9-byte body, got %d", len(out))
	}
	if out[0] != 1 {
		t.Fatalf("expected flag byte=1, got %d", out[0])
	}
}

func TestSetCardEncodeShapeNotAdded(t *testing.T) {
	body := SetCard{CardId: 2380000, Level: 3, Added: false}
	out := body.Encode(logrus.New(), context.Background())(map[string]interface{}{})
	if out[0] != 0 {
		t.Fatalf("expected flag byte=0, got %d", out[0])
	}
}
