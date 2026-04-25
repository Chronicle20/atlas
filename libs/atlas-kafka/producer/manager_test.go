package producer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"
)

// fakeWriter is a Writer implementation used by manager tests. Tracks how many
// times Close was called and supports an injected close error.
type fakeWriter struct {
	topicName string
	closeErr  error
	closes    int32
}

func (f *fakeWriter) Topic() string { return f.topicName }
func (f *fakeWriter) WriteMessages(_ context.Context, _ ...kafka.Message) error {
	return nil
}
func (f *fakeWriter) Close() error {
	atomic.AddInt32(&f.closes, 1)
	return f.closeErr
}

func TestManager_LazyCreate(t *testing.T) {
	ResetInstance()
	var built int32
	factory := func(topicName string) Writer {
		atomic.AddInt32(&built, 1)
		return &fakeWriter{topicName: topicName}
	}
	m := GetManager(ConfigWriterFactory(factory))
	l, _ := test.NewNullLogger()

	w1, err := m.Writer(l, "MY_TOPIC")
	if err != nil {
		t.Fatalf("first Writer call returned error: %v", err)
	}
	w2, err := m.Writer(l, "MY_TOPIC")
	if err != nil {
		t.Fatalf("second Writer call returned error: %v", err)
	}
	if w1 != w2 {
		t.Fatalf("expected same Writer instance on repeat lookup; got distinct pointers")
	}
	if got := atomic.LoadInt32(&built); got != 1 {
		t.Fatalf("factory should be called exactly once; got %d", got)
	}
}

// Suppress unused-import warning until later tests reference these.
var _ = sync.Once{}
var _ = errors.New
