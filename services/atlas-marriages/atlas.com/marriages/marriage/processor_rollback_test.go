package marriage

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// Proposal acceptance updates the proposal row then creates the marriage row
// (class A: proposals + marriages). Failing the marriage create must roll the
// proposal back to pending.
func TestAcceptProposal_RollsBackProposalUpdateWhenMarriageCreateFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)

	prop := ProposalEntity{
		ProposerId: 1001,
		TargetId:   1002,
		Status:     ProposalStatusPending,
		ProposedAt: time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
		TenantId:   tid,
	}
	require.NoError(t, db.Create(&prop).Error)

	databasetest.FailWritesOn(t, db, "marriages", databasetest.WriteCreate)

	l, _ := test.NewNullLogger()
	_, err := NewProcessor(l, ctx, db).AcceptProposalAndEmit(uuid.New(), prop.ID)
	require.Error(t, err)

	var after ProposalEntity
	require.NoError(t, db.First(&after, prop.ID).Error)
	require.Equal(t, ProposalStatusPending, after.Status, "proposal update must roll back with the failed marriage create")

	var marriages int64
	require.NoError(t, db.Model(&Entity{}).Count(&marriages).Error)
	require.Zero(t, marriages)
}
