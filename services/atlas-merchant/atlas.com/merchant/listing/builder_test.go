package listing

import (
	"atlas-merchant/kafka/message/asset"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder_ValidListing(t *testing.T) {
	id := uuid.New()
	shopId := uuid.New()
	snapshot := asset.AssetData{Flag: 1}

	m, err := NewBuilder().
		SetId(id).
		SetShopId(shopId).
		SetItemId(2000000).
		SetItemType(2).
		SetQuantity(100).
		SetBundleSize(10).
		SetBundlesRemaining(10).
		SetPricePerBundle(5000).
		SetItemSnapshot(snapshot).
		SetDisplayOrder(0).
		SetListedAt(time.Now()).
		Build()

	require.NoError(t, err)
	assert.Equal(t, id, m.Id())
	assert.Equal(t, shopId, m.ShopId())
	assert.Equal(t, uint32(2000000), m.ItemId())
	assert.Equal(t, byte(2), m.ItemType())
	assert.Equal(t, uint16(100), m.Quantity())
	assert.Equal(t, uint16(10), m.BundleSize())
	assert.Equal(t, uint16(10), m.BundlesRemaining())
	assert.Equal(t, uint32(5000), m.PricePerBundle())
}

func TestBuilder_MissingId(t *testing.T) {
	_, err := NewBuilder().
		SetShopId(uuid.New()).
		SetPricePerBundle(100).
		SetBundleSize(1).
		Build()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

func TestBuilder_MissingShopId(t *testing.T) {
	_, err := NewBuilder().
		SetId(uuid.New()).
		SetPricePerBundle(100).
		SetBundleSize(1).
		Build()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "shopId is required")
}

func TestBuilder_ZeroPrice(t *testing.T) {
	_, err := NewBuilder().
		SetId(uuid.New()).
		SetShopId(uuid.New()).
		SetPricePerBundle(0).
		SetBundleSize(1).
		Build()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pricePerBundle must be at least 1")
}

func TestBuilder_ZeroBundleSize(t *testing.T) {
	_, err := NewBuilder().
		SetId(uuid.New()).
		SetShopId(uuid.New()).
		SetPricePerBundle(100).
		SetBundleSize(0).
		Build()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bundleSize must be at least 1")
}

func TestBuilder_SingleItemBundle(t *testing.T) {
	m, err := NewBuilder().
		SetId(uuid.New()).
		SetShopId(uuid.New()).
		SetItemId(1000000).
		SetBundleSize(1).
		SetBundlesRemaining(5).
		SetQuantity(5).
		SetPricePerBundle(10000).
		Build()
	require.NoError(t, err)
	assert.Equal(t, uint16(1), m.BundleSize())
	assert.Equal(t, uint16(5), m.BundlesRemaining())
}

func TestClone(t *testing.T) {
	original, err := NewBuilder().
		SetId(uuid.New()).
		SetShopId(uuid.New()).
		SetItemId(2000000).
		SetItemType(2).
		SetQuantity(50).
		SetBundleSize(5).
		SetBundlesRemaining(10).
		SetPricePerBundle(3000).
		Build()
	require.NoError(t, err)

	cloned, err := Clone(original).
		SetBundlesRemaining(8).
		SetQuantity(40).
		Build()
	require.NoError(t, err)

	assert.Equal(t, original.Id(), cloned.Id())
	assert.Equal(t, uint16(8), cloned.BundlesRemaining())
	assert.Equal(t, uint16(40), cloned.Quantity())
	// Original unchanged
	assert.Equal(t, uint16(10), original.BundlesRemaining())
}
