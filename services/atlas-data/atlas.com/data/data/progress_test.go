package data

import (
	"atlas-data/data/workers"
	"context"
	"errors"
	"fmt"
	"testing"
)

type progressEvent struct {
	name    string
	state   string
	errText string
}

type progressRecordingSink struct{ events []progressEvent }

func (s *progressRecordingSink) WorkerStarted(_ context.Context, name string) {
	s.events = append(s.events, progressEvent{name: name, state: "running"})
}

func (s *progressRecordingSink) WorkerFinished(_ context.Context, name string, err error, skipped bool) {
	e := progressEvent{name: name, state: "succeeded"}
	switch {
	case skipped:
		e.state = "skipped"
	case err != nil:
		e.state = "failed"
		e.errText = err.Error()
	}
	s.events = append(s.events, e)
}

func TestRunWithProgressSucceeded(t *testing.T) {
	s := &progressRecordingSink{}
	err := runWithProgress(context.Background(), s, "MAP", func(context.Context) error { return nil })
	if err != nil {
		t.Fatalf("got %v, want nil", err)
	}
	want := []progressEvent{{name: "MAP", state: "running"}, {name: "MAP", state: "succeeded"}}
	if fmt.Sprint(s.events) != fmt.Sprint(want) {
		t.Fatalf("events = %v, want %v", s.events, want)
	}
}

func TestRunWithProgressFailed(t *testing.T) {
	s := &progressRecordingSink{}
	boom := errors.New("boom")
	err := runWithProgress(context.Background(), s, "MAP", func(context.Context) error { return boom })
	if !errors.Is(err, boom) {
		t.Fatalf("error not propagated: %v", err)
	}
	if len(s.events) != 2 || s.events[1].state != "failed" || s.events[1].errText != "boom" {
		t.Fatalf("events = %v", s.events)
	}
}

// A category genuinely absent from a monolithic Data.wz (v12 has no Quest)
// records the worker as skipped and returns nil, so the run can still succeed.
func TestRunWithProgressSkipsCategoryAbsent(t *testing.T) {
	s := &progressRecordingSink{}
	err := runWithProgress(context.Background(), s, "QUEST", func(context.Context) error {
		return fmt.Errorf("QUEST open Quest.wz: %w", workers.ErrCategoryAbsent)
	})
	if err != nil {
		t.Fatalf("got %v, want nil (a skipped worker must not fail the run)", err)
	}
	if len(s.events) != 2 || s.events[1].state != "skipped" {
		t.Fatalf("events = %v, want the second to be skipped", s.events)
	}
	if s.events[1].errText != "" {
		t.Fatalf("skipped worker carries error %q", s.events[1].errText)
	}
}

func TestNewRunConfigDefaultsToNoopSink(t *testing.T) {
	if _, ok := newRunConfig(nil).sink.(noopSink); !ok {
		t.Fatal("default sink is not noopSink")
	}
	if _, ok := newRunConfig([]RunOption{WithProgress(nil)}).sink.(noopSink); !ok {
		t.Fatal("WithProgress(nil) replaced the default sink")
	}
	s := &progressRecordingSink{}
	if got := newRunConfig([]RunOption{WithProgress(s)}).sink; got != ProgressSink(s) {
		t.Fatalf("WithProgress did not install the sink, got %T", got)
	}
}
