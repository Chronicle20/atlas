package merchant

import (
	"testing"

	merchant2 "atlas-channel/kafka/message/merchant"

	"github.com/stretchr/testify/assert"

	interactioncb "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/clientbound"
)

func TestShopCreateFailureMode(t *testing.T) {
	assert.Equal(t, interactioncb.CharacterInteractionEnterErrorModeCannotOpenStoreNearPortal, shopCreateFailureMode(merchant2.ShopCreateFailReasonTooCloseToPortal))
	assert.Equal(t, interactioncb.CharacterInteractionEnterErrorModeCannotOpenMiniRoomHere, shopCreateFailureMode(merchant2.ShopCreateFailReasonTooCloseToShop))
	assert.Equal(t, interactioncb.CharacterInteractionEnterErrorModeMustBeInFreeMarket, shopCreateFailureMode(merchant2.ShopCreateFailReasonNotFreeMarket))
	assert.Equal(t, interactioncb.CharacterInteractionEnterErrorModeUnable, shopCreateFailureMode(merchant2.ShopCreateFailReasonUnable))
	assert.Equal(t, interactioncb.CharacterInteractionEnterErrorModeUnable, shopCreateFailureMode("SOMETHING_ELSE"))
}
