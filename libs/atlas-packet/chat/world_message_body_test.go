package chat

import (
	"context"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
)

// worldMessageOptions mirrors the tenant WorldMessage writer's options.operations
// map (docs/packets/... seed templates carry MEGAPHONE/SUPER_MEGAPHONE/
// ITEM_MEGAPHONE/MULTI_MEGAPHONE keys). Values are float64 as ResolveCode
// decodes JSON-number-typed config (resolve.go).
func worldMessageOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			"MEGAPHONE":       float64(2),
			"SUPER_MEGAPHONE": float64(3),
			"ITEM_MEGAPHONE":  float64(8),
			"MULTI_MEGAPHONE": float64(10),
		},
	}
}

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

// TestWorldMessageBodyResolvesMode confirms each per-mode body function
// config-resolves its leading mode byte from the "operations" table (DOM-25)
// rather than a hard-coded literal.
func TestWorldMessageBodyResolvesMode(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	opts := worldMessageOptions()

	cases := []struct {
		name string
		emit func() []byte
		want byte
	}{
		{"megaphone", func() []byte {
			return WorldMessageMegaphoneBody("hello")(l, ctx)(opts)
		}, 2},
		{"super megaphone", func() []byte {
			return WorldMessageSuperMegaphoneBody("hello", 0, false)(l, ctx)(opts)
		}, 3},
		{"item megaphone", func() []byte {
			return WorldMessageItemMegaphoneBody("hello", 0, false, nil)(l, ctx)(opts)
		}, 8},
		{"multi megaphone", func() []byte {
			return WorldMessageMultiMegaphoneBody([]string{"a", "b"}, 0, false)(l, ctx)(opts)
		}, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.emit()
			if len(got) == 0 {
				t.Fatalf("%s: empty output", c.name)
			}
			if got[0] != c.want {
				t.Fatalf("%s: leading mode byte = %d, want %d (99 = unresolved)", c.name, got[0], c.want)
			}
		})
	}
}

// TestWorldMessageBodyUnconfiguredModeDegrades confirms a mode missing from
// the tenant "operations" table degrades to 99 (ResolveCode's documented
// misconfiguration sentinel) rather than panicking or silently sending 0.
func TestWorldMessageBodyUnconfiguredModeDegrades(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	opts := map[string]interface{}{
		"operations": map[string]interface{}{
			"MEGAPHONE": float64(2),
		},
	}

	cases := []struct {
		name string
		emit func() []byte
	}{
		{"super megaphone", func() []byte {
			return WorldMessageSuperMegaphoneBody("hello", 0, false)(l, ctx)(opts)
		}},
		{"item megaphone", func() []byte {
			return WorldMessageItemMegaphoneBody("hello", 0, false, nil)(l, ctx)(opts)
		}},
		{"multi megaphone", func() []byte {
			return WorldMessageMultiMegaphoneBody([]string{"a"}, 0, false)(l, ctx)(opts)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.emit()
			if len(got) == 0 {
				t.Fatalf("%s: empty output", c.name)
			}
			if got[0] != 99 {
				t.Fatalf("%s: leading mode byte = %d, want 99 (unresolved degrade)", c.name, got[0])
			}
		})
	}
}
