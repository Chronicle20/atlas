package service

import (
	"context"
	"errors"
	"io"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

type fakeProjection struct {
	mu           sync.Mutex
	startGroupId string
	startCtx     context.Context
	startErr     error
	waitFn       func(ctx context.Context) error
}

func (f *fakeProjection) Start(ctx context.Context, _ logrus.FieldLogger, _ *sync.WaitGroup, groupId string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.startCtx = ctx
	f.startGroupId = groupId
	return f.startErr
}

func (f *fakeProjection) WaitCaughtUp(ctx context.Context) error {
	if f.waitFn != nil {
		return f.waitFn(ctx)
	}
	return nil
}

func TestWithConfigProjectionStartsSubscriberWithGeneratedGroupId(t *testing.T) {
	t.Setenv("EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS", "svc-status")
	t.Setenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS", "tenant-status")
	fake := &fakeProjection{}
	var gotTopics ProjectionTopics
	rt := Bootstrap("atlas-test", WithoutTracer(),
		WithConfigProjection("Test Service - abc", func(topics ProjectionTopics) Projection {
			gotTopics = topics
			return fake
		}),
	)
	if gotTopics.ServiceStatus != "svc-status" || gotTopics.TenantStatus != "tenant-status" {
		t.Fatalf("topics not resolved from env: %+v", gotTopics)
	}
	want := regexp.MustCompile(`^Test Service - abc - projection - [0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !want.MatchString(fake.startGroupId) {
		t.Fatalf("groupId %q does not match per-process pattern", fake.startGroupId)
	}
	if fake.startCtx == nil {
		t.Fatal("Start not bound to teardown context")
	}
	rt.AwaitProjectionCatchUp() // fake catches up immediately; must return
}

func TestAwaitProjectionCatchUpTimeoutFatal(t *testing.T) {
	t.Setenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS", "tenant-status")
	t.Setenv("PROJECTION_CATCHUP_TIMEOUT_S", "1")
	fake := &fakeProjection{waitFn: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	rt := Bootstrap("atlas-test", WithoutTracer(),
		WithConfigProjection("Test Service", func(ProjectionTopics) Projection { return fake }),
	)
	rt.Logger().SetOutput(io.Discard)
	exited := false
	rt.Logger().ExitFunc = func(int) { exited = true; panic("exit") }
	func() {
		defer func() { _ = recover() }()
		rt.AwaitProjectionCatchUp()
	}()
	if !exited {
		t.Fatal("catch-up timeout must Fatal (process exit)")
	}
}

func TestAwaitProjectionCatchUpWithoutOptionPanics(t *testing.T) {
	rt := Bootstrap("atlas-test", WithoutTracer())
	defer func() {
		if recover() == nil {
			t.Fatal("AwaitProjectionCatchUp without WithConfigProjection must panic")
		}
	}()
	rt.AwaitProjectionCatchUp()
}

func TestProjectionFuncsAdapts(t *testing.T) {
	var startedGroup string
	waitErr := errors.New("nope")
	p := ProjectionFuncs{
		StartFunc: func(_ context.Context, _ logrus.FieldLogger, _ *sync.WaitGroup, g string) error {
			startedGroup = g
			return nil
		},
		WaitCaughtUpFunc: func(context.Context) error { return waitErr },
	}
	if err := p.Start(context.Background(), nil, nil, "g1"); err != nil || startedGroup != "g1" {
		t.Fatalf("Start delegation broken: %v %q", err, startedGroup)
	}
	if !errors.Is(p.WaitCaughtUp(context.Background()), waitErr) {
		t.Fatal("WaitCaughtUp delegation broken")
	}
}

func TestParseProjectionCatchupTimeout(t *testing.T) {
	tests := []struct {
		val  string
		want time.Duration
	}{
		{"", 5 * time.Minute},
		{"30", 30 * time.Second},
		{"0", 5 * time.Minute},
		{"-4", 5 * time.Minute},
		{"garbage", 5 * time.Minute},
	}
	for _, tc := range tests {
		t.Setenv("PROJECTION_CATCHUP_TIMEOUT_S", tc.val)
		if got := parseProjectionCatchupTimeout(); got != tc.want {
			t.Errorf("PROJECTION_CATCHUP_TIMEOUT_S=%q → %v, want %v", tc.val, got, tc.want)
		}
	}
}
