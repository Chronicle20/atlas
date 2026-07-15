package shop

import (
	"testing"

	asset2 "atlas-merchant/kafka/message/asset"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WithdrawMeso credits the hired merchant's accrued balance to the owner and
// zeroes it; a non-owner is rejected; a personal shop has no meso to withdraw.
func TestWithdrawMeso(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	setupTestRegistries(t)
	p := NewProcessor(l, ctx, db).(*ProcessorImpl)

	const owner, stranger = uint32(7000), uint32(7001)
	m, err := p.CreateShop(owner, HiredMerchant, "Merch", 0, 0, 910000001, uuid.Nil, 0, 0, 5030000)
	require.NoError(t, err)
	require.NoError(t, db.WithContext(ctx).Model(&Entity{}).Where("id = ?", m.Id()).Updates(map[string]interface{}{"state": byte(Open), "meso_balance": uint32(50000)}).Error)

	mb := testBuffer()

	// Non-owner rejected.
	err = p.WithdrawMeso(mb)(m.Id(), stranger)
	assert.ErrorIs(t, err, ErrNotOwner)

	// Owner withdraws: balance zeroed.
	require.NoError(t, p.WithdrawMeso(mb)(m.Id(), owner))
	got, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, uint32(0), got.MesoBalance())
}

// OrganizeListings drops sold-out rows and compacts display orders; an empty
// result closes the shop.
func TestOrganizeListings(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	setupTestRegistries(t)
	p := NewProcessor(l, ctx, db).(*ProcessorImpl)

	const owner = uint32(7100)
	m, err := p.CreateShop(owner, HiredMerchant, "Merch", 0, 0, 910000001, uuid.Nil, 0, 0, 5030000)
	require.NoError(t, err)

	mb := testBuffer()
	snap := asset2.AssetData{Quantity: 1}
	// Three listings added during setup (Draft); middle one then sold out.
	_, err = p.AddListing(mb)(m.Id(), owner, 2000000, 2, 1, 1, 100, snap, 2, 1)
	require.NoError(t, err)
	sold, err := p.AddListing(mb)(m.Id(), owner, 2000001, 2, 1, 1, 100, snap, 2, 2)
	require.NoError(t, err)
	_, err = p.AddListing(mb)(m.Id(), owner, 2000002, 2, 1, 1, 100, snap, 2, 3)
	require.NoError(t, err)
	require.NoError(t, db.WithContext(ctx).Table("listings").Where("id = ?", sold.Id()).Update("bundles_remaining", 0).Error)
	require.NoError(t, db.WithContext(ctx).Model(&Entity{}).Where("id = ?", m.Id()).Update("state", byte(Open)).Error)

	require.NoError(t, p.OrganizeListings(mb)(m.Id(), owner))

	listings, err := p.GetListings(m.Id())
	require.NoError(t, err)
	require.Len(t, listings, 2, "sold-out listing removed")
	orders := []uint16{listings[0].DisplayOrder(), listings[1].DisplayOrder()}
	assert.Equal(t, []uint16{0, 1}, orders, "display orders compacted")
}
