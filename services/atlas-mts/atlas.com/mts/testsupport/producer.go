// Package testsupport holds the env-gated MTS test routes
// (design-e2e-testing.md): data seeding, listing time travel, an on-demand
// expiration sweep, and simulated buyer/bidder actions. The simulated actions
// emit the SAME Kafka commands the channel emits for a real client, so the
// full production path (consumer -> processor -> saga -> orchestrator ->
// wallet/custody) runs. Routes are registered only when
// MTS_TEST_ROUTES_ENABLED=true and are never routed through ingress.
package testsupport

import (
	mtsmsg "atlas-mts/kafka/message/mts"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// BuyCommandProvider mirrors the channel's BuyCommandProvider
// (services/atlas-channel/.../mts/producer.go) field-for-field: buyer-keyed,
// CommandBuy envelope. Keep the two in lockstep — the fidelity of /test
// purchases rests on this being indistinguishable from a client-driven buy.
func BuyCommandProvider(transactionId uuid.UUID, worldId world.Id, serial uint32, buyerId uint32, buyerAccountId uint32, buyNow bool) model.Provider[[]kafka.Message] {
	key := kprod.CreateKey(int(buyerId))
	value := &mtsmsg.Command[mtsmsg.BuyCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandBuy,
		Body: mtsmsg.BuyCommandBody{
			WorldId:        byte(worldId),
			Serial:         serial,
			BuyerId:        buyerId,
			BuyerAccountId: buyerAccountId,
			BuyNow:         buyNow,
		},
	}
	return kprod.SingleMessageProvider(key, value)
}

// PlaceBidCommandProvider mirrors the channel's PlaceBidCommandProvider:
// bidder-keyed, CommandPlaceBid envelope carrying the raw bid amount (the
// consumer applies the commission mark-up at escrow time, same as for a
// client-driven bid).
func PlaceBidCommandProvider(transactionId uuid.UUID, worldId world.Id, serial uint32, bidderId uint32, bidderAccountId uint32, amount uint32) model.Provider[[]kafka.Message] {
	key := kprod.CreateKey(int(bidderId))
	value := &mtsmsg.Command[mtsmsg.PlaceBidCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandPlaceBid,
		Body: mtsmsg.PlaceBidCommandBody{
			WorldId:         byte(worldId),
			Serial:          serial,
			BidderId:        bidderId,
			BidderAccountId: bidderAccountId,
			Amount:          amount,
		},
	}
	return kprod.SingleMessageProvider(key, value)
}
