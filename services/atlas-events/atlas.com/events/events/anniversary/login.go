package anniversary

import (
	"atlas-events/event/occurrence"
	"atlas-events/kafka/message"
	"atlas-events/kafka/message/buff"
	"atlas-events/kafka/message/characterstatus"
	"context"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// LoginProcessor grants every active ANNIVERSARY occurrence's buff to a
// character entering gameplay (FR-A7).
type LoginProcessor interface {
	OnLogin(e characterstatus.StatusEvent[characterstatus.StatusEventLoginBody]) error
}

// LoginProcessorImpl is the LoginProcessor implementation. Mirrors Handler's
// (l, ctx, db) storage shape.
type LoginProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

// NewLoginProcessor constructs a LoginProcessorImpl.
func NewLoginProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *LoginProcessorImpl {
	return &LoginProcessorImpl{l: l, ctx: ctx, db: db}
}

// compile-time assertion
var _ LoginProcessor = (*LoginProcessorImpl)(nil)

// OnLogin grants the active occurrence's buff to a character entering
// gameplay (FR-A7). This is a REACTION, not a query in the login path:
// atlas-events being unavailable delays the buff, it never delays or fails
// the login (FR-A8).
//
// The buff is applied with noExpiry: the occurrence — not a duration — is
// the authoritative fact that Anniversary is happening (FR-A5), and
// completion cancels it explicitly by correlation (FR-A15) via
// cancelByCorrelationCommandProvider in handler.go, which cancels by the
// SAME occurrence.Id().String() this emits as CorrelationId (R34-6).
func (p *LoginProcessorImpl) OnLogin(e characterstatus.StatusEvent[characterstatus.StatusEventLoginBody]) error {
	os, err := occurrence.NewProcessor(p.l, p.ctx, p.db).GetActiveByType(TypeName)
	if err != nil {
		return err
	}
	for _, o := range os {
		c, err := DecodeOccurrenceContext(o.Context())
		if err != nil {
			return err
		}
		if err := p.emitApply(e.WorldId, e.Body.ChannelId, e.CharacterId, o.Id(), c); err != nil {
			return err
		}
	}
	return nil
}

// emitApply emits a single APPLY command granting occurrenceId's buff to
// characterId.
func (p *LoginProcessorImpl) emitApply(worldId world.Id, channelId channel.Id, characterId uint32, occurrenceId uuid.UUID, c OccurrenceContext) error {
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		return buf.Put(buff.EnvCommandTopic, applyCommandProvider(worldId, channelId, characterId, occurrenceId, c))
	})
}

// applyCommandProvider builds the APPLY command carrying the occurrence's
// two multipliers, scaled by 100 (2.0x -> amount 200, per ConversionDirect,
// Task 8). NoExpiry: true / Duration: 0 — the atlas-buffs consumer rejects
// NoExpiry with a nonzero duration.
func applyCommandProvider(worldId world.Id, channelId channel.Id, characterId uint32, occurrenceId uuid.UUID, c OccurrenceContext) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buff.Command[buff.ApplyCommandBody]{
		WorldId:     worldId,
		ChannelId:   channelId,
		CharacterId: characterId,
		Type:        buff.CommandTypeApply,
		Body: buff.ApplyCommandBody{
			SourceId: c.BuffSourceId,
			Duration: 0,
			NoExpiry: true,
			Changes: []buff.StatChange{
				{Type: string(charconst.TemporaryStatTypeExpBuffRate), Amount: int32(c.ExpMultiplier * 100)},
				{Type: string(charconst.TemporaryStatTypeItemUpByItem), Amount: int32(c.DropMultiplier * 100)},
			},
			CorrelationId: occurrenceId.String(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
