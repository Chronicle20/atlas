package shop

import (
	"testing"

	asset2 "atlas-merchant/kafka/message/asset"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shop mutations arrive over the Kafka command surface, which trusts the
// characterId the producer supplies — the server must therefore verify that
// the actor owns the shop for every owner-only mutation, not rely on the
// channel's gating alone.
func TestOwnerMutations_RejectNonOwner(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	setupTestRegistries(t)
	p := NewProcessor(l, ctx, db).(*ProcessorImpl)

	const owner, stranger = uint32(5000), uint32(5001)
	m, err := p.CreateShop(owner, CharacterShop, "Owned", 0, 0, 910000001, uuid.Nil, 0, 0, 5140000)
	require.NoError(t, err)

	mb := testBuffer()

	_, err = p.AddListing(mb)(m.Id(), stranger, 2000000, 2, 1, 1, 100, asset2.AssetData{Quantity: 1}, 2, 1)
	assert.ErrorIs(t, err, ErrNotOwner, "AddListing by non-owner")

	err = p.OpenShop(mb)(m.Id(), stranger)
	assert.ErrorIs(t, err, ErrNotOwner, "OpenShop by non-owner")

	_, err = p.RemoveListing(mb)(m.Id(), stranger, 0)
	assert.ErrorIs(t, err, ErrNotOwner, "RemoveListing by non-owner")

	err = p.CloseShop(mb)(m.Id(), stranger, CloseReasonManualClose)
	assert.ErrorIs(t, err, ErrNotOwner, "CloseShop by non-owner")

	// Force Open for the maintenance transitions.
	require.NoError(t, db.WithContext(ctx).Model(&Entity{}).Where("id = ?", m.Id()).Update("state", byte(Open)).Error)
	err = p.EnterMaintenance(mb)(m.Id(), stranger)
	assert.ErrorIs(t, err, ErrNotOwner, "EnterMaintenance by non-owner")

	require.NoError(t, db.WithContext(ctx).Model(&Entity{}).Where("id = ?", m.Id()).Update("state", byte(Maintenance)).Error)
	err = p.ExitMaintenance(mb)(m.Id(), stranger)
	assert.ErrorIs(t, err, ErrNotOwner, "ExitMaintenance by non-owner")

	// The owner can still close (system reapers pass the owner id).
	err = p.CloseShop(mb)(m.Id(), owner, CloseReasonManualClose)
	assert.NoError(t, err, "CloseShop by owner")
}
