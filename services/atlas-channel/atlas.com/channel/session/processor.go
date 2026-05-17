package session

import (
	"atlas-channel/account/session"
	session2 "atlas-channel/kafka/message/session"
	"atlas-channel/kafka/producer"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"net"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	socketpkt "github.com/Chronicle20/atlas/libs/atlas-packet/socket/clientbound"
	socket "github.com/Chronicle20/atlas/libs/atlas-socket"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	kp  producer.Provider
	sp  session.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		kp:  producer.ProviderImpl(l)(ctx),
		sp:  session.NewProcessor(l, ctx),
	}
	return p
}

func (p *Processor) WithContext(ctx context.Context) *Processor {
	return NewProcessor(p.l, ctx)
}

func (p *Processor) AllInTenantProvider() ([]Model, error) {
	return getRegistry().GetInTenant(p.t.Id()), nil
}

func (p *Processor) AllInChannelProvider(worldId world.Id, channelId channel.Id) ([]Model, error) {
	all := getRegistry().GetInTenant(p.t.Id())
	result := make([]Model, 0)
	for _, s := range all {
		if s.WorldId() == worldId && s.ChannelId() == channelId {
			result = append(result, s)
		}
	}
	return result, nil
}

func (p *Processor) ByIdModelProvider(sessionId uuid.UUID) model.Provider[Model] {
	t := tenant.MustFromContext(p.ctx)
	return func() (Model, error) {
		s, ok := getRegistry().Get(t.Id(), sessionId)
		if !ok {
			return Model{}, errors.New("not found")
		}
		return s, nil
	}
}

func (p *Processor) IfPresentById(sessionId uuid.UUID, f model.Operator[Model]) {
	s, err := p.ByIdModelProvider(sessionId)()
	if err != nil {
		return
	}
	_ = f(s)
}

func (p *Processor) IfPresentByIdInWorld(sessionId uuid.UUID, ch channel.Model, f model.Operator[Model]) {
	s, err := p.ByIdModelProvider(sessionId)()
	if err != nil {
		return
	}
	if s.WorldId() != ch.WorldId() {
		return
	}
	if s.ChannelId() != ch.Id() {
		return
	}
	_ = f(s)
}

func (p *Processor) ByCharacterIdModelProvider(ch channel.Model) func(characterId uint32) model.Provider[Model] {
	return func(characterId uint32) model.Provider[Model] {
		return model.FirstProvider[Model](p.AllInTenantProvider, model.Filters(CharacterIdFilter(characterId), WorldIdFilter(ch.WorldId()), ChannelIdFilter(ch.Id())))
	}
}

// IfPresentByCharacterId executes an Operator if a session exists for the characterId
func (p *Processor) IfPresentByCharacterId(ch channel.Model) func(characterId uint32, f model.Operator[Model]) error {
	return func(characterId uint32, f model.Operator[Model]) error {
		s, err := p.ByCharacterIdModelProvider(ch)(characterId)()
		if err != nil {
			return nil
		}
		return f(s)
	}
}

func CharacterIdFilter(referenceId uint32) model.Filter[Model] {
	return func(model Model) bool {
		return model.CharacterId() == referenceId
	}
}

func AccountIdFilter(accountId uint32) model.Filter[Model] {
	return func(model Model) bool {
		return model.AccountId() == accountId
	}
}

func WorldIdFilter(worldId world.Id) model.Filter[Model] {
	return func(model Model) bool {
		return model.WorldId() == worldId
	}
}

func ChannelIdFilter(channelId channel.Id) model.Filter[Model] {
	return func(model Model) bool {
		return model.ChannelId() == channelId
	}
}

func (p *Processor) ByAccountIdModelProvider(ch channel.Model) func(accountId uint32) model.Provider[Model] {
	return func(accountId uint32) model.Provider[Model] {
		return model.FirstProvider[Model](p.AllInTenantProvider, model.Filters(AccountIdFilter(accountId), WorldIdFilter(ch.WorldId()), ChannelIdFilter(ch.Id())))
	}
}

// IfPresentByAccountId executes an Operator if a session exists for the accountId
func (p *Processor) IfPresentByAccountId(ch channel.Model) func(accountId uint32, f model.Operator[Model]) error {
	return func(accountId uint32, f model.Operator[Model]) error {
		s, err := p.ByAccountIdModelProvider(ch)(accountId)()
		if err != nil {
			return nil
		}
		return f(s)
	}
}

// GetByCharacterId gets a session (if one exists) for the given characterId
func (p *Processor) GetByCharacterId(ch channel.Model) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		return p.ByCharacterIdModelProvider(ch)(characterId)()
	}
}

func (p *Processor) ForEachByCharacterId(ch channel.Model) func(provider model.Provider[[]uint32], f model.Operator[Model]) error {
	return func(provider model.Provider[[]uint32], f model.Operator[Model]) error {
		return model.ForEachSlice(model.SliceMap[uint32, Model](p.GetByCharacterId(ch))(provider)(), f, model.ParallelExecute())
	}
}

func Announce(l logrus.FieldLogger) func(ctx context.Context) func(writerProducer writer.Producer) func(writerName string) func(encoder packet.Encode) model.Operator[Model] {
	return func(ctx context.Context) func(writerProducer writer.Producer) func(writerName string) func(encoder packet.Encode) model.Operator[Model] {
		return func(writerProducer writer.Producer) func(writerName string) func(encoder packet.Encode) model.Operator[Model] {
			return func(writerName string) func(encoder packet.Encode) model.Operator[Model] {
				return func(encoder packet.Encode) model.Operator[Model] {
					return func(s Model) error {
						spanCtx, span := otel.GetTracerProvider().Tracer("atlas-channel").Start(ctx, "session.Announce")
						defer span.End()
						t := tenant.MustFromContext(ctx)
						span.SetAttributes(
							attribute.String("writer.name", writerName),
							attribute.String("tenant.id", t.Id().String()),
							attribute.Int("world.id", int(s.WorldId())),
						)

						w, err := writerProducer(writerName)
						if err != nil {
							span.RecordError(err)
							span.SetStatus(codes.Error, err.Error())
							return err
						}
						if err := s.announceEncrypted(w(l, spanCtx)(encoder)); err != nil {
							span.RecordError(err)
							span.SetStatus(codes.Error, err.Error())
							return err
						}
						return nil
					}
				}
			}
		}
	}
}

func (p *Processor) SetAccountId(id uuid.UUID, accountId uint32) Model {
	s := Model{}
	var ok bool
	if s, ok = getRegistry().Get(p.t.Id(), id); ok {
		s = s.setAccountId(accountId)
		getRegistry().Update(p.t.Id(), s)
		return s
	}
	return s
}

func (p *Processor) SetCharacterId(id uuid.UUID, characterId uint32) Model {
	s := Model{}
	var ok bool
	if s, ok = getRegistry().Get(p.t.Id(), id); ok {
		s = s.setCharacterId(characterId)
		getRegistry().Update(p.t.Id(), s)
		return s
	}
	return s
}

func (p *Processor) SetMapId(id uuid.UUID, mapId _map.Id) Model {
	s := Model{}
	var ok bool
	if s, ok = getRegistry().Get(p.t.Id(), id); ok {
		s = s.setMapId(mapId)
		getRegistry().Update(p.t.Id(), s)
		return s
	}
	return s
}

func (p *Processor) SetField(id uuid.UUID, f field.Model) Model {
	s := Model{}
	var ok bool
	if s, ok = getRegistry().Get(p.t.Id(), id); ok {
		s = s.setMapId(f.MapId())
		s = s.setInstance(f.Instance())
		getRegistry().Update(p.t.Id(), s)
		return s
	}
	return s
}

func (p *Processor) SetGm(id uuid.UUID, gm bool) Model {
	s := Model{}
	var ok bool
	if s, ok = getRegistry().Get(p.t.Id(), id); ok {
		s = s.setGm(gm)
		getRegistry().Update(p.t.Id(), s)
		return s
	}
	return s
}

func (p *Processor) UpdateLastRequest(id uuid.UUID) Model {
	s := Model{}
	var ok bool
	if s, ok = getRegistry().Get(p.t.Id(), id); ok {
		s = s.updateLastRequest()
		getRegistry().Update(p.t.Id(), s)
		return s
	}
	return s
}

func (p *Processor) SessionCreated(s Model) error {
	return p.kp(session2.EnvEventTopicSessionStatus)(CreatedStatusEventProvider(s.SessionId(), s.AccountId(), s.CharacterId(), s.Field().Channel()))
}

func Teardown(l logrus.FieldLogger) func() {
	return func() {
		ctx, span := otel.GetTracerProvider().Tracer("atlas-channel").Start(context.Background(), "teardown")
		defer span.End()

		_ = tenant.ForAll(func(t tenant.Model) error {
			p := NewProcessor(l, tenant.WithContext(ctx, t))
			return model.ForEachSlice(p.AllInTenantProvider, p.Destroy)
		})
	}
}

func (p *Processor) Create(ch channel.Model, locale byte) func(sessionId uuid.UUID, conn net.Conn) {
	return func(sessionId uuid.UUID, conn net.Conn) {
		fl := p.l.WithField("session", sessionId)
		fl.Debugf("Creating session.")
		s := NewSession(sessionId, p.t, locale, conn)
		s = s.setWorldId(ch.WorldId())
		s = s.setChannelId(ch.Id())
		getRegistry().Add(p.t.Id(), s)

		err := s.WriteHello(p.t.MajorVersion(), p.t.MinorVersion())
		if err != nil {
			fl.WithError(err).Errorf("Unable to write hello packet.")
		}
	}
}

func (p *Processor) Decrypt(hasAes bool, hasMapleEncryption bool) func(sessionId uuid.UUID, input []byte) []byte {
	return func(sessionId uuid.UUID, input []byte) []byte {
		s, ok := getRegistry().Get(p.t.Id(), sessionId)
		if !ok {
			return input
		}
		if s.ReceiveAESOFB() == nil {
			return input
		}
		return s.ReceiveAESOFB().Decrypt(hasAes, hasMapleEncryption)(input)
	}
}

func (p *Processor) DestroyByIdWithSpan(sessionId uuid.UUID) {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-channel").Start(p.ctx, "session-destroy")
	defer span.End()
	p.WithContext(sctx).DestroyById(sessionId)
}

func (p *Processor) DestroyById(sessionId uuid.UUID) {
	s, ok := getRegistry().Get(p.t.Id(), sessionId)
	if !ok {
		return
	}
	_ = p.Destroy(s)
}

func (p *Processor) Destroy(s Model) error {
	p.l.WithField("session", s.SessionId().String()).Debugf("Destroying session.")
	getRegistry().Remove(p.t.Id(), s.SessionId())

	// Emit logout and destroyed events BEFORE closing the socket so a
	// crash-safe ordering exists: a downstream consumer that sees the
	// destroyed event can no longer race with the socket-close path
	// (FR-CHN-14). The two emit failures are demoted from hard error to
	// logged warning so a flaky producer can't strand the connection in
	// a half-closed state; the final Disconnect always runs.
	p.sp.Destroy(s.SessionId(), s.AccountId())
	emitErr := p.kp(session2.EnvEventTopicSessionStatus)(DestroyedStatusEventProvider(s.SessionId(), s.AccountId(), s.CharacterId(), s.Field().Channel()))
	if emitErr != nil {
		p.l.WithError(emitErr).Warn("session.destroy.emit_destroyed_failed")
	}

	s.Disconnect()
	return emitErr
}

func (p *Processor) SetStorageNpcId(id uuid.UUID, npcId uint32) Model {
	s := Model{}
	var ok bool
	if s, ok = getRegistry().Get(p.t.Id(), id); ok {
		s = s.setStorageNpcId(npcId)
		getRegistry().Update(p.t.Id(), s)
		return s
	}
	return s
}

func (p *Processor) ClearStorageNpcId(id uuid.UUID) Model {
	s := Model{}
	var ok bool
	if s, ok = getRegistry().Get(p.t.Id(), id); ok {
		s = s.clearStorageNpcId()
		getRegistry().Update(p.t.Id(), s)
		return s
	}
	return s
}

func SendPing(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) socket.IdleNotifier {
	t := tenant.MustFromContext(ctx)
	return func(sessionId uuid.UUID) {
		s, ok := getRegistry().Get(t.Id(), sessionId)
		if !ok {
			return
		}
		l.Debugf("Session [%s] idle, sending PING.", sessionId)
		_ = Announce(l)(ctx)(wp)(socketpkt.PingWriter)(socketpkt.Ping{}.Encode)(s)
	}
}
