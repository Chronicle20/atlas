package invite

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

// envMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since invite sits outside env-domain-guard's permitted
// import list (main.go, kafka/, rest/, socket/) and must not import
// atlas-env even from a test file.
type envMarkerKey string

// TestProcessExpiredInvitesAppliesEnvContextToDeleteAndReject pins the
// review fix: this pod's own environment identity must be threaded onto the
// context passed to both del and reject for each expired invite. Without
// this, decide() would fail open per FR-1.8 and every live deployment, not
// just this pod's, would act on the expired invite.
func TestProcessExpiredInvitesAppliesEnvContextToDeleteAndReject(t *testing.T) {
	l := setupTestLogger(t)
	ten := setupTestTenant(t)
	i, err := NewBuilder().
		SetTenant(ten).
		SetId(1).
		SetInviteType("BUDDY").
		SetOriginatorId(100).
		SetTargetId(200).
		Build()
	if err != nil {
		t.Fatalf("failed to build invite: %v", err)
	}

	envContext := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, envMarkerKey("marker"), "stamped")
	}

	var delMarker any
	var rejectMarker any
	processExpiredInvites(l, context.Background(), []Model{i},
		func(_ logrus.FieldLogger, ctx context.Context, _ Model) error {
			delMarker = ctx.Value(envMarkerKey("marker"))
			return nil
		},
		func(_ logrus.FieldLogger, ctx context.Context, _ Model) error {
			rejectMarker = ctx.Value(envMarkerKey("marker"))
			return nil
		},
		envContext,
	)

	if delMarker != "stamped" {
		t.Fatalf("envContext was not applied to the delete context: got %v, want \"stamped\"", delMarker)
	}
	if rejectMarker != "stamped" {
		t.Fatalf("envContext was not applied to the reject context: got %v, want \"stamped\"", rejectMarker)
	}
}

// TestProcessExpiredInvitesSkipsRejectOnDeleteFailure pins the existing
// fail-open bookkeeping behavior: a failed delete must not be followed by a
// reject emit for the same invite.
func TestProcessExpiredInvitesSkipsRejectOnDeleteFailure(t *testing.T) {
	l := setupTestLogger(t)
	ten := setupTestTenant(t)
	i, err := NewBuilder().
		SetTenant(ten).
		SetId(1).
		SetInviteType("BUDDY").
		SetOriginatorId(100).
		SetTargetId(200).
		Build()
	if err != nil {
		t.Fatalf("failed to build invite: %v", err)
	}

	rejectCalled := false
	processExpiredInvites(l, context.Background(), []Model{i},
		func(_ logrus.FieldLogger, _ context.Context, _ Model) error {
			return context.DeadlineExceeded
		},
		func(_ logrus.FieldLogger, _ context.Context, _ Model) error {
			rejectCalled = true
			return nil
		},
		func(ctx context.Context) context.Context { return ctx },
	)

	if rejectCalled {
		t.Fatalf("reject must not be called when delete fails")
	}
}
