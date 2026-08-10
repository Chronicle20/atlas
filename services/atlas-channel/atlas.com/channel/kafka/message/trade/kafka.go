// Package trade carries the COMMAND_TOPIC_TRADE / EVENT_TOPIC_TRADE_STATUS
// envelopes. Mirrors
// services/atlas-trades/atlas.com/trades/kafka/message/trade/kafka.go;
// struct names, field names and json tags must match that file exactly.
package trade

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic     = "COMMAND_TOPIC_TRADE"
	EnvEventTopicStatus = "EVENT_TOPIC_TRADE_STATUS"

	CommandTypeCreateRoom    = "CREATE_ROOM"
	CommandTypeInvite        = "INVITE"
	CommandTypeDeclineInvite = "DECLINE_INVITE"
	CommandTypeEnterRoom     = "ENTER_ROOM"
	CommandTypePutItem       = "PUT_ITEM"
	CommandTypeAddMeso       = "ADD_MESO"
	CommandTypeConfirm       = "CONFIRM"
	CommandTypeTransaction   = "TRANSACTION"
	CommandTypeCancel        = "CANCEL"
	CommandTypeChat          = "CHAT"

	StatusTypeRoomCreated          = "ROOM_CREATED"
	StatusTypeInviteSent           = "INVITE_SENT"
	StatusTypeInviteRejected       = "INVITE_REJECTED"
	StatusTypeParticipantEntered   = "PARTICIPANT_ENTERED"
	StatusTypeItemStaged           = "ITEM_STAGED"
	StatusTypeItemRefused          = "ITEM_REFUSED"
	StatusTypeMesoStaged           = "MESO_STAGED"
	StatusTypeMesoRefused          = "MESO_REFUSED"
	StatusTypeParticipantConfirmed = "PARTICIPANT_CONFIRMED"
	StatusTypeAttestationRequested = "ATTESTATION_REQUESTED"
	StatusTypeSettled              = "SETTLED"
	StatusTypeCancelled            = "CANCELLED"
	StatusTypeError                = "ERROR"
	StatusTypeChat                 = "CHAT"
)

// Command is the atlas-channel -> atlas-trades envelope. COMMAND_TOPIC_TRADE is
// shared by every handler registered on it, so a handler MUST discriminate on
// Type before unmarshalling Body into a concrete command body.
type Command[E any] struct {
	TransactionId uuid.UUID    `json:"transactionId"`
	WorldId       world.Id     `json:"worldId"`
	ChannelId     channel.Id   `json:"channelId"`
	MapId         _map.Id      `json:"mapId"`
	Instance      uuid.UUID    `json:"instance"`
	CharacterId   character.Id `json:"characterId"`
	Type          string       `json:"type"`
	Body          E            `json:"body"`
}

type CreateRoomCommandBody struct {
	RoomType byte `json:"roomType"`
}

type InviteCommandBody struct {
	TargetCharacterId character.Id `json:"targetCharacterId"`
}

type DeclineInviteCommandBody struct {
	SerialNumber uint32 `json:"serialNumber"`
	ErrorCode    byte   `json:"errorCode"`
}

// EnterRoomCommandBody names the room by its wire handle AND the room type the
// caller believes it is entering. The handle is the owner's character id
// (design §2.3) and therefore public, so it is not on its own an admission
// ticket: atlas-trades additionally requires the enterer to be the character
// the room's outstanding invite named, and the room's kind to match RoomType.
type EnterRoomCommandBody struct {
	Handle   uint32 `json:"handle"`
	RoomType byte   `json:"roomType"`
}

// PutItemCommandBody mirrors the serverbound OperationTradePutItem decode.
// Quantity stays uint16 — the client field is an Encode2, and widening it to
// asset.Quantity would misrepresent the decoded width at the boundary.
type PutItemCommandBody struct {
	InventoryType inventory.Type `json:"inventoryType"`
	Slot          slot.Position  `json:"slot"`
	Quantity      uint16         `json:"quantity"`
	TargetSlot    byte           `json:"targetSlot"`
}

// AddMesoCommandBody carries the ABSOLUTE total from the client's input box,
// not a delta (CTradingRoomDlg::PutMoney, design §1.6). Signed because the
// serverbound codec is Encode4 of a signed int32 and a hostile client can send
// a negative.
type AddMesoCommandBody struct {
	Amount int32 `json:"amount"`
}

// CrcEntry is one {data, crc} pair from a TRADE_CONFIRM or TRANSACTION
// payload. Absent on GMS <= v79 (tradeCrcPresent), where the lists are empty.
type CrcEntry struct {
	Data uint32 `json:"data"`
	Crc  uint32 `json:"crc"`
}

type ConfirmCommandBody struct {
	Entries []CrcEntry `json:"entries"`
}

type TransactionCommandBody struct {
	Entries []CrcEntry `json:"entries"`
}

type CancelCommandBody struct{}

type ChatCommandBody struct {
	Message string `json:"message"`
}

// StatusEvent is the atlas-trades -> atlas-channel envelope. It always carries
// both participants so the channel can address the room without a lookup.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID    `json:"transactionId"`
	WorldId       world.Id     `json:"worldId"`
	ChannelId     channel.Id   `json:"channelId"`
	MapId         _map.Id      `json:"mapId"`
	Instance      uuid.UUID    `json:"instance"`
	RoomId        uuid.UUID    `json:"roomId"`
	Handle        uint32       `json:"handle"`
	RoomType      byte         `json:"roomType"`
	OwnerId       character.Id `json:"ownerId"`
	VisitorId     character.Id `json:"visitorId"`
	CharacterId   character.Id `json:"characterId"`
	Type          string       `json:"type"`
	Body          E            `json:"body"`
}

type RoomCreatedEventBody struct {
	// Position is the recipient's seat: 0 owner, 1 visitor (FR-1.5).
	Position byte `json:"position"`
}

type ParticipantEnteredEventBody struct {
	CharacterId character.Id `json:"characterId"`
	Name        string       `json:"name"`
	Position    byte         `json:"position"`
}

type InviteSentEventBody struct {
	TargetCharacterId character.Id `json:"targetCharacterId"`
	InviterName       string       `json:"inviterName"`
}

// ItemStagedEventBody names the staging SIDE by position, not by character.
// atlas-channel converts it to each recipient's own recipient-relative side
// byte before writing the packet.
type ItemStagedEventBody struct {
	Position      byte           `json:"position"`
	TradeSlot     byte           `json:"tradeSlot"`
	InventoryType inventory.Type `json:"inventoryType"`
	SourceSlot    slot.Position  `json:"sourceSlot"`
	AssetId       asset.Id       `json:"assetId"`
	TemplateId    item.Id        `json:"templateId"`
	Quantity      asset.Quantity `json:"quantity"`
}

type MesoStagedEventBody struct {
	Position byte   `json:"position"`
	Amount   uint32 `json:"amount"`
}

// MesoRefusedEventBody drives the authoritative re-echo: atlas-channel sends
// TRADE_ADD_MESO with LastValidAmount so the client's view snaps back, plus
// TRADE_MESO_LIMIT where that arm exists (design §4.2).
type MesoRefusedEventBody struct {
	Position        byte   `json:"position"`
	LastValidAmount uint32 `json:"lastValidAmount"`
}

// ItemRefusedEventBody is the item twin of MesoRefusedEventBody, and it exists
// for the LOCK rather than for the picture.
//
// A refused stage is player-visibly silent — the empty trade slot is the whole
// feedback (design §7) — but the client armed CWvsContext::m_bExclRequestSent
// when it sent PUT_ITEM, and CanSendExclRequest then refuses every subsequent
// exclusive request, including ADD_MESO, until a server packet clears it. A
// silent drop therefore wedges the dialog for the rest of the session. This
// event is what atlas-channel turns into that unlock (design §5A.6).
//
// TradeSlot is carried for diagnostics: the refusal renders nothing, so the
// client never needs to know which slot it was.
type ItemRefusedEventBody struct {
	Position  byte `json:"position"`
	TradeSlot byte `json:"tradeSlot"`
}

type ParticipantConfirmedEventBody struct {
	Position byte `json:"position"`
}

type AttestationRequestedEventBody struct{}

type SettledEventBody struct {
	LedgerEntryId uuid.UUID `json:"ledgerEntryId"`
}

// CancelledEventBody carries the semantic leaveReason KEY string, which
// atlas-channel resolves to a per-version status byte via the tenant
// leaveReason table (DOM-25). Never a numeric status.
type CancelledEventBody struct {
	Reason string `json:"reason"`
}

// InviteRejectedEventBody carries an inviteResult KEY string, which
// atlas-channel resolves to a per-version code via the tenant inviteResult
// table (DOM-25), plus the refused target's name.
//
// TargetName is load-bearing, not decoration: GMS v83
// CMiniRoomBaseDlg::OnInviteResultStatic (@0x65E848) reads no name for code 1
// (CANNOT_FIND_CHARACTER) but DecodeStr's one for codes 2/3/4 and Formats it
// into the message ("%s is doing something else right now"). An empty name on
// a BUSY refusal renders that message with a blank subject. It is therefore
// empty ONLY for a refusal whose arm reads no name, or when the target could
// not be read at all — which is itself always reported as CANNOT_FIND_CHARACTER.
type InviteRejectedEventBody struct {
	Code       string `json:"code"`
	TargetName string `json:"targetName"`
}

// ErrorEventBody carries an enterError KEY string, resolved the same way.
type ErrorEventBody struct {
	Code string `json:"code"`
}

type ChatEventBody struct {
	Position byte   `json:"position"`
	Message  string `json:"message"`
}
