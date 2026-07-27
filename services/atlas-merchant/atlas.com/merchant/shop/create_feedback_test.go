package shop

import (
	"testing"

	merchant "atlas-merchant/kafka/message/merchant"

	"github.com/stretchr/testify/assert"
)

func TestShopCreateFailureReason(t *testing.T) {
	assert.Equal(t, merchant.ShopCreateFailReasonTooCloseToPortal, shopCreateFailureReason(ErrTooCloseToPortal))
	assert.Equal(t, merchant.ShopCreateFailReasonTooCloseToShop, shopCreateFailureReason(ErrTooCloseToShop))
	assert.Equal(t, merchant.ShopCreateFailReasonNotFreeMarket, shopCreateFailureReason(ErrNotFreemarketRoom))
	assert.Equal(t, merchant.ShopCreateFailReasonUnable, shopCreateFailureReason(ErrShopLimitReached))
	assert.Equal(t, merchant.ShopCreateFailReasonUnable, shopCreateFailureReason(ErrFrederickPending))
	// Internal errors get no player-facing reason.
	assert.Equal(t, "", shopCreateFailureReason(ErrNotFound))
}
