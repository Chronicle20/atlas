package tenantpurge

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// fakeNamespaceLister returns a configurable namespace deletion timestamp.
type fakeNamespaceLister struct {
	delTs *time.Time
	err   error
	calls int
}

func (f *fakeNamespaceLister) NamespaceDeletionTimestamp(ctx context.Context, name string) (*time.Time, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.delTs, nil
}

// fakeTenantEnumerator returns a fixed list of tenant ids.
type fakeTenantEnumerator struct {
	ids   []string
	err   error
	calls int
}

func (f *fakeTenantEnumerator) EnumerateTenants(ctx context.Context) ([]string, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.ids, nil
}

// TestPurgeAllIfNamespaceTerminating_NoDeletion verifies a routine
// restart (namespace deletionTimestamp not set) is a no-op.
func TestPurgeAllIfNamespaceTerminating_NoDeletion(t *testing.T) {
	l, _ := loggerForTest()
	nsLister := &fakeNamespaceLister{delTs: nil}
	tenantEnum := &fakeTenantEnumerator{ids: []string{"a8657a40-4dc9-4d7a-a0d2-3a9edb69d141"}}

	PurgeAllIfNamespaceTerminating(
		context.Background(), l, nil, nil, nsLister, tenantEnum, "atlas-pr-99",
	)

	if nsLister.calls != 1 {
		t.Fatalf("expected NamespaceDeletionTimestamp called once; got %d", nsLister.calls)
	}
	if tenantEnum.calls != 0 {
		t.Fatalf("expected EnumerateTenants NOT called when namespace not being deleted; got %d calls", tenantEnum.calls)
	}
}

// TestPurgeAllIfNamespaceTerminating_NoLister verifies the function is
// safe to call with nil lister (e.g., kubernetes client unavailable).
func TestPurgeAllIfNamespaceTerminating_NoLister(t *testing.T) {
	l, _ := loggerForTest()
	tenantEnum := &fakeTenantEnumerator{ids: []string{"a8657a40-4dc9-4d7a-a0d2-3a9edb69d141"}}

	PurgeAllIfNamespaceTerminating(
		context.Background(), l, nil, nil, nil, tenantEnum, "atlas-pr-99",
	)

	if tenantEnum.calls != 0 {
		t.Fatalf("expected EnumerateTenants NOT called with nil nsLister; got %d", tenantEnum.calls)
	}
}

// TestPurgeAllIfNamespaceTerminating_LookupError treats kube-API errors
// as "unknown deletion state" and skips the purge — better to leak
// than to wipe data on a transient API outage.
func TestPurgeAllIfNamespaceTerminating_LookupError(t *testing.T) {
	l, _ := loggerForTest()
	nsLister := &fakeNamespaceLister{err: errors.New("api down")}
	tenantEnum := &fakeTenantEnumerator{ids: []string{"a8657a40-4dc9-4d7a-a0d2-3a9edb69d141"}}

	PurgeAllIfNamespaceTerminating(
		context.Background(), l, nil, nil, nsLister, tenantEnum, "atlas-pr-99",
	)

	if tenantEnum.calls != 0 {
		t.Fatalf("expected EnumerateTenants NOT called when namespace lookup errors; got %d calls", tenantEnum.calls)
	}
}

// TestPurgeAllIfNamespaceTerminating_DeletionSet verifies tenant
// enumeration is invoked when deletionTimestamp is set. We pass nil
// db/mc so the actual Purge call panics — but EnumerateTenants is
// called first, which is the contract we're asserting.
func TestPurgeAllIfNamespaceTerminating_DeletionSet(t *testing.T) {
	l, _ := loggerForTest()
	now := time.Now()
	nsLister := &fakeNamespaceLister{delTs: &now}
	// Return empty list so Purge is never called (we don't have a real DB).
	tenantEnum := &fakeTenantEnumerator{ids: []string{}}

	PurgeAllIfNamespaceTerminating(
		context.Background(), l, nil, nil, nsLister, tenantEnum, "atlas-pr-99",
	)

	if tenantEnum.calls != 1 {
		t.Fatalf("expected EnumerateTenants called once when namespace is being deleted; got %d", tenantEnum.calls)
	}
}

// TestPurgeAllIfNamespaceTerminating_InvalidUUID confirms a malformed
// tenant id is logged and skipped (not crash).
func TestPurgeAllIfNamespaceTerminating_InvalidUUID(t *testing.T) {
	l, hook := loggerForTest()
	now := time.Now()
	nsLister := &fakeNamespaceLister{delTs: &now}
	tenantEnum := &fakeTenantEnumerator{ids: []string{"not-a-uuid"}}

	// Should not panic.
	PurgeAllIfNamespaceTerminating(
		context.Background(), l, nil, nil, nsLister, tenantEnum, "atlas-pr-99",
	)

	// Expect a warn log mentioning the invalid id.
	found := false
	for _, e := range hook.AllEntries() {
		if e.Level <= logrus.WarnLevel && contains(e.Message, "invalid tenant id") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected warn log about invalid tenant id; got:\n%v", hook.AllEntries())
	}
}

// Helpers ------------------------------------------------------------

// loggerForTest returns a logrus.Logger + test hook capturing entries.
func loggerForTest() (*logrus.Logger, *testHook) {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	hook := &testHook{}
	l.AddHook(hook)
	return l, hook
}

type testHook struct {
	entries []*logrus.Entry
}

func (h *testHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *testHook) Fire(e *logrus.Entry) error {
	h.entries = append(h.entries, e)
	return nil
}

func (h *testHook) AllEntries() []*logrus.Entry {
	return h.entries
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
