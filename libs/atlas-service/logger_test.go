package service

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestLoggerEmitsTheEnvironmentField(t *testing.T) {
	t.Setenv("ATLAS_ENVIRONMENT", "pr-123")
	var buf bytes.Buffer
	l := CreateLogger("atlas-test")
	l.SetOutput(&buf)
	l.Info("hello")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec["environment"] != "pr-123" {
		t.Fatalf("environment = %v, want \"pr-123\"", rec["environment"])
	}
}

func TestLoggerOmitsTheEnvironmentFieldOnMain(t *testing.T) {
	// NFR-7: main's log records are byte-identical to today's.
	t.Setenv("ATLAS_ENVIRONMENT", "")
	var buf bytes.Buffer
	l := CreateLogger("atlas-test")
	l.SetOutput(&buf)
	l.Info("hello")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, present := rec["environment"]; present {
		t.Fatal("environment field emitted with no ATLAS_ENVIRONMENT set")
	}
}

func TestCreateLoggerEmitsServiceNameAndNormalizedKeys(t *testing.T) {
	l := CreateLogger("atlas-test")
	var buf bytes.Buffer
	l.SetOutput(&buf)

	l.WithField("characterId", 42).Info("hello")

	out := buf.String()
	if !strings.Contains(out, "character_id") {
		t.Errorf("emitted record missing normalized key: %s", out)
	}
	if strings.Contains(out, "characterId") {
		t.Errorf("emitted record still contains camelCase key: %s", out)
	}
	if !strings.Contains(out, "atlas-test") {
		t.Errorf("emitted record missing service name: %s", out)
	}
}

func TestCreateLoggerLogLevelEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	if l := CreateLogger("atlas-test"); l.GetLevel() != logrus.DebugLevel {
		t.Errorf("LOG_LEVEL=debug not honored, got %v", l.GetLevel())
	}
	t.Setenv("LOG_LEVEL", "not-a-level")
	if l := CreateLogger("atlas-test"); l.GetLevel() != logrus.InfoLevel {
		t.Errorf("invalid LOG_LEVEL must silently keep the default, got %v", l.GetLevel())
	}
}

// Pin the logrus v1.9.4 safety property the normalizer relies on: hooks fire
// on a per-emission copy (entry.Dup()), so a shared derived entry logged
// from parallel goroutines does not race the in-place key rewrite.
func TestCreateLoggerSharedEntryParallelEmitNoRace(t *testing.T) {
	l := CreateLogger("atlas-test")
	l.SetOutput(&safeBuffer{})
	e := l.WithField("characterId", 42)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				e.Info("parallel")
			}
		}()
	}
	wg.Wait()
}

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}
