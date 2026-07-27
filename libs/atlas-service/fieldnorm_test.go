package service

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNormalizeFieldKey(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		changed bool
	}{
		{"characterId", "character_id", true},
		{"characterID", "character_id", true},
		{"transactionId", "transaction_id", true},
		{"worldId2", "world_id2", true},
		{"HTTPServer", "http_server", true},
		{"Name", "name", true},
		{"character_id", "character_id", false}, // already snake
		{"tenant", "tenant", false},             // plain lowercase
		{"service.name", "service.name", false}, // dotted ECS key passes through
		{"ms.Version", "ms.Version", false},     // any dotted key passes through, even with uppercase
	}
	for _, tc := range tests {
		got, changed := normalizeFieldKey(tc.in)
		if got != tc.want || changed != tc.changed {
			t.Errorf("normalizeFieldKey(%q) = (%q, %v), want (%q, %v)", tc.in, got, changed, tc.want, tc.changed)
		}
	}
}

func fireNormalizer(t *testing.T, data logrus.Fields) logrus.Fields {
	t.Helper()
	entry := &logrus.Entry{Data: data}
	if err := (fieldKeyNormalizerHook{}).Fire(entry); err != nil {
		t.Fatal(err)
	}
	return entry.Data
}

func TestNormalizerHookRewritesKeys(t *testing.T) {
	got := fireNormalizer(t, logrus.Fields{"characterId": 42, "world_id": 1, "service.name": "x"})
	if got["character_id"] != 42 {
		t.Errorf("character_id = %v, want 42", got["character_id"])
	}
	if _, ok := got["characterId"]; ok {
		t.Error("camelCase key survived")
	}
	if got["world_id"] != 1 || got["service.name"] != "x" {
		t.Errorf("passthrough keys damaged: %v", got)
	}
}

func TestNormalizerHookCollisionSnakeCaseWins(t *testing.T) {
	got := fireNormalizer(t, logrus.Fields{"characterId": 1, "character_id": 2})
	if got["character_id"] != 2 {
		t.Errorf("collision: character_id = %v, want the explicit snake_case value 2", got["character_id"])
	}
	if len(got) != 1 {
		t.Errorf("expected exactly 1 key after collision, got %v", got)
	}
}

func TestNormalizerHookIdempotent(t *testing.T) {
	data := logrus.Fields{"characterId": 42}
	first := fireNormalizer(t, data)
	second := fireNormalizer(t, first)
	if second["character_id"] != 42 || len(second) != 1 {
		t.Errorf("second pass changed data: %v", second)
	}
}
