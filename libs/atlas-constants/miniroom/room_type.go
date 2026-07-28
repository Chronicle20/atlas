// Package miniroom holds the shared mini-room (CMiniRoom) type discriminator
// byte that identifies which kind of room a character has opened. The value is
// the leading byte of every mini-room create/enter/balloon packet and of the
// mini-game room/record Kafka events. It was previously triplicated as a local
// const block in atlas-mini-games' game processor, the interaction.RoomType
// constants in atlas-packet, and atlas-channel's gameTypeCode helper; those
// three sites now derive from these constants.
package miniroom

// The mini-room type bytes are grounded in the client's CMiniRoom subclass
// discriminator (IDA / Cosmic parity): Omok and MatchCards are the two
// mini-games; Trade through CashTrade are the shop/trade rooms.
const (
	Omok         byte = 1
	MatchCards   byte = 2
	Trade        byte = 3
	PersonalShop byte = 4
	MerchantShop byte = 5
	CashTrade    byte = 6
)
