package coupon

import (
	"atlas-cashshop/kafka/message/cashshop"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// CouponRedeemedStatusEventProvider builds the event for a COMMITTED
// redemption. maplePoints and credit are the DELTAS this coupon awarded, not
// balances (see cashshop.CouponRedeemedBody).
func CouponRedeemedStatusEventProvider(characterId uint32, compartmentId uuid.UUID, assetIds []uint32, maplePoints uint32, credit uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.CouponRedeemedBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeCouponRedeemed,
		Body: cashshop.CouponRedeemedBody{
			CompartmentId: compartmentId,
			AssetIds:      assetIds,
			MaplePoints:   maplePoints,
			Credit:        credit,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// CouponFailedStatusEventProvider builds the event for a redemption that
// changed nothing. errorKey is one of the coupon.ErrorKey* values.
func CouponFailedStatusEventProvider(characterId uint32, errorKey string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.CouponFailedBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeCouponFailed,
		Body:        cashshop.CouponFailedBody{Error: errorKey},
	}
	return producer.SingleMessageProvider(key, value)
}
