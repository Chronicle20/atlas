package trade

import (
	trade2 "atlas-channel/kafka/message/trade"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor is a thin fire-and-forget producer onto COMMAND_TOPIC_TRADE, plus
// the one read atlas-channel needs of atlas-trades' room registry.
// atlas-channel never mutates inventory or meso for trade — all trade state
// lives in atlas-trades (task-205 design §2.2).
type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

// MemberModelProvider retrieves the (0-or-1) trade room characterId currently
// occupies (owner or invitee) from atlas-trades.
func (p *Processor) MemberModelProvider(characterId character.Id) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByMember(p.ctx, characterId), Extract, model.Filters[Model]())
}

// InGame reports whether characterId is currently seated in a trade room. It is
// the trade half of the cross-family occupancy check the interaction handler
// runs before opening a mini-room (design §2.1); atlas-trades still enforces
// its own single-room invariant authoritatively.
func (p *Processor) InGame(characterId character.Id) (bool, error) {
	rooms, err := p.MemberModelProvider(characterId)()
	if err != nil {
		return false, err
	}
	return len(rooms) > 0, nil
}

func (p *Processor) CreateRoom(f field.Model, characterId character.Id, roomType byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(trade2.EnvCommandTopic)(CreateRoomCommandProvider(uuid.New(), f, characterId, roomType))
}

func (p *Processor) Invite(f field.Model, characterId character.Id, targetCharacterId character.Id) error {
	return producer.ProviderImpl(p.l)(p.ctx)(trade2.EnvCommandTopic)(InviteCommandProvider(uuid.New(), f, characterId, targetCharacterId))
}

// DeclineInvite forwards the serial the client echoed back. The serial is a
// wire handle, not a character id — atlas-trades resolves the room from it.
func (p *Processor) DeclineInvite(f field.Model, characterId character.Id, serialNumber uint32, errorCode byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(trade2.EnvCommandTopic)(DeclineInviteCommandProvider(uuid.New(), f, characterId, serialNumber, errorCode))
}

// EnterRoom names the room by the handle the client sent and the room type its
// dialog claimed. Neither is trusted: atlas-trades admits only the character
// its outstanding invite named, and only into a room of the claimed kind.
func (p *Processor) EnterRoom(f field.Model, characterId character.Id, handle uint32, roomType byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(trade2.EnvCommandTopic)(EnterRoomCommandProvider(uuid.New(), f, characterId, handle, roomType))
}

// PutItem takes an already-validated inventory.Type: the wire field is an
// unsigned byte and inventory.Type is a SIGNED int8, so the interaction
// handler narrows and range-checks it at the decode boundary before calling
// here (tradeInventoryType).
func (p *Processor) PutItem(f field.Model, characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position, quantity uint16, targetSlot byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(trade2.EnvCommandTopic)(PutItemCommandProvider(uuid.New(), f, characterId, inventoryType, sourceSlot, quantity, targetSlot))
}

// AddMeso carries the ABSOLUTE total from the client's input box, not a delta,
// and stays signed because a hostile client can send a negative.
func (p *Processor) AddMeso(f field.Model, characterId character.Id, amount int32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(trade2.EnvCommandTopic)(AddMesoCommandProvider(uuid.New(), f, characterId, amount))
}

func (p *Processor) Confirm(f field.Model, characterId character.Id, entries []trade2.CrcEntry) error {
	return producer.ProviderImpl(p.l)(p.ctx)(trade2.EnvCommandTopic)(ConfirmCommandProvider(uuid.New(), f, characterId, entries))
}

func (p *Processor) Transaction(f field.Model, characterId character.Id, entries []trade2.CrcEntry) error {
	return producer.ProviderImpl(p.l)(p.ctx)(trade2.EnvCommandTopic)(TransactionCommandProvider(uuid.New(), f, characterId, entries))
}

func (p *Processor) Cancel(f field.Model, characterId character.Id) error {
	return producer.ProviderImpl(p.l)(p.ctx)(trade2.EnvCommandTopic)(CancelCommandProvider(uuid.New(), f, characterId))
}

func (p *Processor) Chat(f field.Model, characterId character.Id, message string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(trade2.EnvCommandTopic)(ChatCommandProvider(uuid.New(), f, characterId, message))
}
