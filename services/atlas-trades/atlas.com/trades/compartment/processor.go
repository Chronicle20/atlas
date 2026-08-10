package compartment

import (
	"atlas-trades/kafka/message"
	compartmentmsg "atlas-trades/kafka/message/compartment"
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// ErrQuantityOutOfRange is returned when a caller asks to reserve a quantity
// the wire cannot carry. The reserve command body's quantity is int16
// (atlas-inventory's ItemBody.Quantity), so zero and anything above
// math.MaxInt16 are rejected HERE rather than silently wrapping into a
// negative that atlas-inventory would widen to a near-4-billion reservation.
var ErrQuantityOutOfRange = errors.New("compartment: reserve quantity out of the wire's int16 range")

// Processor issues the inventory reservation commands trade staging depends on.
// Every method takes the caller's message buffer so the commands land in the
// same transactional-outbox batch as the trade status events they accompany.
type Processor interface {
	// RequestReserve holds `quantity` of the asset in `sourceSlot` for
	// `expiry`, under the reservation handle `reservationId`.
	RequestReserve(mb *message.Buffer) func(reservationId uuid.UUID, characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position, templateId item.Id, quantity asset.Quantity, expiry time.Duration) error
	// CancelReservation releases the reservation `reservationId` holds on
	// (inventoryType, sourceSlot).
	CancelReservation(mb *message.Buffer) func(reservationId uuid.UUID, characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) RequestReserve(mb *message.Buffer) func(reservationId uuid.UUID, characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position, templateId item.Id, quantity asset.Quantity, expiry time.Duration) error {
	return func(reservationId uuid.UUID, characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position, templateId item.Id, quantity asset.Quantity, expiry time.Duration) error {
		if quantity == 0 || quantity > math.MaxInt16 {
			return ErrQuantityOutOfRange
		}
		p.l.Debugf("Reserving [%d] of item [%d] in compartment [%d] slot [%d] for character [%d] under reservation [%s].", quantity, templateId, inventoryType, sourceSlot, characterId, reservationId.String())
		return mb.Put(compartmentmsg.EnvCommandTopic, requestReserveCommandProvider(reservationId, characterId, inventoryType, sourceSlot, templateId, int16(quantity), expiry))
	}
}

func (p *ProcessorImpl) CancelReservation(mb *message.Buffer) func(reservationId uuid.UUID, characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position) error {
	return func(reservationId uuid.UUID, characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position) error {
		p.l.Debugf("Cancelling reservation [%s] on compartment [%d] slot [%d] for character [%d].", reservationId.String(), inventoryType, sourceSlot, characterId)
		return mb.Put(compartmentmsg.EnvCommandTopic, cancelReservationCommandProvider(reservationId, characterId, inventoryType, sourceSlot))
	}
}
