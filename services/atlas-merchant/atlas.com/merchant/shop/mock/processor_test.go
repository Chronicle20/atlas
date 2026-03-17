package mock

import (
	"atlas-merchant/listing"
	"atlas-merchant/shop"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProcessorMock_DefaultBehavior(t *testing.T) {
	m := &ProcessorMock{}

	id := uuid.New()

	model, err := m.GetById(id)
	assert.NoError(t, err)
	assert.Equal(t, shop.Model{}, model)

	models, err := m.GetByCharacterId(1000)
	assert.NoError(t, err)
	assert.Empty(t, models)

	models, err = m.GetByField(0, 0, 100, uuid.Nil)
	assert.NoError(t, err)
	assert.Empty(t, models)

	listings, err := m.GetListings(id)
	assert.NoError(t, err)
	assert.Empty(t, listings)

	created, err := m.CreateShop(1000, shop.CharacterShop, "Test", 0, 0, 100, uuid.Nil, 0, 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, shop.Model{}, created)

	assert.NoError(t, m.OpenShop(id))
	assert.NoError(t, m.EnterMaintenance(id))

	closed, err := m.ExitMaintenance(id)
	assert.NoError(t, err)
	assert.False(t, closed)

	assert.NoError(t, m.CloseShop(id, shop.CloseReasonManualClose))

	expired, err := m.GetExpired()
	assert.NoError(t, err)
	assert.Empty(t, expired)

	li, err := m.AddListing(id, 2000000, 0, 1, 1, 1000, nil, 0)
	assert.NoError(t, err)
	assert.Equal(t, listing.Model{}, li)

	li, err = m.RemoveListing(id, 0)
	assert.NoError(t, err)
	assert.Equal(t, listing.Model{}, li)

	assert.NoError(t, m.UpdateListing(id, 0, 1000, 1, 1))
	assert.NoError(t, m.EnterShop(1000, id))
	assert.NoError(t, m.ExitShop(1000, id))

	ejected, err := m.EjectAllVisitors(id)
	assert.NoError(t, err)
	assert.Nil(t, ejected)

	visitors, err := m.GetVisitors(id)
	assert.NoError(t, err)
	assert.Nil(t, visitors)

	visitingId, err := m.GetShopForCharacter(1000)
	assert.NoError(t, err)
	assert.Equal(t, uuid.Nil, visitingId)

	result, err := m.PurchaseBundle(2000, id, 0, 1)
	assert.NoError(t, err)
	assert.Equal(t, shop.PurchaseResult{}, result)

	assert.NoError(t, m.OpenShopAndEmit(id, 1000))
	assert.NoError(t, m.CloseShopAndEmit(id, 1000, shop.CloseReasonManualClose))
	assert.NoError(t, m.EnterMaintenanceAndEmit(id, 1000))
	assert.NoError(t, m.ExitMaintenanceAndEmit(id, 1000))
	assert.NoError(t, m.EnterShopAndEmit(1000, id))
	assert.NoError(t, m.ExitShopAndEmit(1000, id))

	result, err = m.PurchaseBundleAndEmit(2000, id, 0, 1, 0)
	assert.NoError(t, err)
	assert.Equal(t, shop.PurchaseResult{}, result)
}

func TestProcessorMock_CustomBehavior(t *testing.T) {
	testErr := errors.New("test error")

	m := &ProcessorMock{
		GetByIdFunc: func(id uuid.UUID) (shop.Model, error) {
			return shop.Model{}, testErr
		},
		CloseShopFunc: func(shopId uuid.UUID, reason shop.CloseReason) error {
			return testErr
		},
		PurchaseBundleFunc: func(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16) (shop.PurchaseResult, error) {
			return shop.PurchaseResult{TotalCost: 5000, Fee: 100}, nil
		},
	}

	_, err := m.GetById(uuid.New())
	assert.ErrorIs(t, err, testErr)

	err = m.CloseShop(uuid.New(), shop.CloseReasonManualClose)
	assert.ErrorIs(t, err, testErr)

	result, err := m.PurchaseBundle(2000, uuid.New(), 0, 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(5000), result.TotalCost)
	assert.Equal(t, int64(100), result.Fee)

	// Non-overridden methods still return defaults.
	assert.NoError(t, m.OpenShop(uuid.New()))
}
