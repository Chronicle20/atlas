package session

import (
	"atlas-login/account/session"
	session2 "atlas-login/kafka/message/session"
	"atlas-login/kafka/producer"
	session3 "atlas-login/kafka/producer/session"
	"atlas-login/socket/writer"
	"context"
	"errors"
	"net"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	socket "github.com/Chronicle20/atlas-socket"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

type Processor interface {
	AllInTenantProvider() ([]Model, error)
	ByIdModelProvider(sessionId uuid.UUID) model.Provider[Model]
	IfPresentById(sessionId uuid.UUID, f model.Operator[Model])
	ByAccountIdModelProvider(accountId uint32) model.Provider[Model]
	IfPresentByAccountId(accountId uint32, f model.Operator[Model]) error
	SetAccountId(id uuid.UUID, accountId uint32) Model
	UpdateLastRequest(id uuid.UUID) Model
	SetWorldId(id uuid.UUID, worldId world.Id) Model
	SetChannelId(id uuid.UUID, channelId channel.Id) Model
	SessionCreated(s Model) error
	Create(locale byte) func(sessionId uuid.UUID, conn net.Conn)
	DestroyByIdWithSpan(sessionId uuid.UUID)
	DestroyById(sessionId uuid.UUID)
	Destroy(s Model) error
	Decrypt(hasAes bool, hasMapleEncryption bool) func(sessionId uuid.UUID, input []byte) []byte
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	kp  producer.Provider
	sp  session.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		kp:  producer.ProviderImpl(l)(ctx),
		sp:  session.NewProcessor(l, ctx),
	}
	return p
}

func (p *ProcessorImpl) WithContext(ctx context.Context) Processor {
	return NewProcessor(p.l, ctx)
}

func (p *ProcessorImpl) AllInTenantProvider() ([]Model, error) {
	return getRegistry().GetInTenant(p.t.Id()), nil
}

func (p *ProcessorImpl) ByIdModelProvider(sessionId uuid.UUID) model.Provider[Model] {
	t := tenant.MustFromContext(p.ctx)
	return func() (Model, error) {
		s, ok := getRegistry().Get(t.Id(), sessionId)
		if !ok {
			return Model{}, errors.New("not found")
		}
		return s, nil
	}
}

func (p *ProcessorImpl) IfPresentById(sessionId uuid.UUID, f model.Operator[Model]) {
	s, err := p.ByIdModelProvider(sessionId)()
	if err != nil {
		return
	}
	_ = f(s)
}

func (p *ProcessorImpl) ByAccountIdModelProvider(accountId uint32) model.Provider[Model] {
	return model.FirstProvider[Model](p.AllInTenantProvider, model.Filters(AccountIdFilter(accountId)))
}

// IfPresentByAccountId executes an Operator if a session exists for the accountId
func (p *ProcessorImpl) IfPresentByAccountId(accountId uint32, f model.Operator[Model]) error {
	s, err := p.ByAccountIdModelProvider(accountId)()
	if err != nil {
		return nil
	}
	return f(s)
}

func AccountIdFilter(referenceId uint32) model.Filter[Model] {
	return func(model Model) bool {
		return model.AccountId() == referenceId
	}
}

func (p *ProcessorImpl) SetAccountId(id uuid.UUID, accountId uint32) Model {
	s := Model{}
	var ok bool
	if s, ok = getRegistry().Get(p.t.Id(), id); ok {
		s = s.setAccountId(accountId)
		getRegistry().Update(p.t.Id(), s)
		return s
	}
	return s
}

func (p *ProcessorImpl) UpdateLastRequest(id uuid.UUID) Model {
	s := Model{}
	var ok bool
	if s, ok = getRegistry().Get(p.t.Id(), id); ok {
		s = s.updateLastRequest()
		getRegistry().Update(p.t.Id(), s)
		return s
	}
	return s
}

func (p *ProcessorImpl) SetWorldId(id uuid.UUID, worldId world.Id) Model {
	s := Model{}
	var ok bool
	if s, ok = getRegistry().Get(p.t.Id(), id); ok {
		s = s.setWorldId(worldId)
		getRegistry().Update(p.t.Id(), s)
		return s
	}
	return s
}

func (p *ProcessorImpl) SetChannelId(id uuid.UUID, channelId channel.Id) Model {
	s := Model{}
	var ok bool
	if s, ok = getRegistry().Get(p.t.Id(), id); ok {
		s = s.setChannelId(channelId)
		getRegistry().Update(p.t.Id(), s)
		return s
	}
	return s
}

func (p *ProcessorImpl) SessionCreated(s Model) error {
	return p.kp(session2.EnvEventTopicSessionStatus)(session3.CreatedStatusEventProvider(s.SessionId(), s.AccountId()))
}

func (p *ProcessorImpl) Create(locale byte) func(sessionId uuid.UUID, conn net.Conn) {
	return func(sessionId uuid.UUID, conn net.Conn) {
		fl := p.l.WithField("session", sessionId)
		fl.Debugf("Creating session.")
		s := NewSession(sessionId, p.t, locale, conn)
		getRegistry().Add(p.t.Id(), s)

		err := s.WriteHello(p.t.MajorVersion(), p.t.MinorVersion())
		if err != nil {
			fl.WithError(err).Errorf("Unable to write hello packet.")
		}
	}
}

func (p *ProcessorImpl) DestroyByIdWithSpan(sessionId uuid.UUID) {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-login").Start(p.ctx, "session-destroy")
	defer span.End()
	p.WithContext(sctx).DestroyById(sessionId)
}

func (p *ProcessorImpl) DestroyById(sessionId uuid.UUID) {
	s, ok := getRegistry().Get(p.t.Id(), sessionId)
	if !ok {
		return
	}
	_ = p.Destroy(s)
}

func (p *ProcessorImpl) Destroy(s Model) error {
	p.l.WithField("session", s.SessionId().String()).Debugf("Destroying session.")
	getRegistry().Remove(p.t.Id(), s.SessionId())
	s.Disconnect()
	p.sp.Destroy(s.SessionId(), s.AccountId())
	return p.kp(session2.EnvEventTopicSessionStatus)(session3.DestroyedStatusEventProvider(s.SessionId(), s.AccountId()))
}

func (p *ProcessorImpl) Decrypt(hasAes bool, hasMapleEncryption bool) func(sessionId uuid.UUID, input []byte) []byte {
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

func Announce(l logrus.FieldLogger) func(writerProducer writer.Producer) func(writerName string) func(s Model, bodyProducer writer.BodyProducer) error {
	return func(writerProducer writer.Producer) func(writerName string) func(s Model, bodyProducer writer.BodyProducer) error {
		return func(writerName string) func(s Model, bodyProducer writer.BodyProducer) error {
			return func(s Model, bodyProducer writer.BodyProducer) error {
				w, err := writerProducer(l, writerName)
				if err != nil {
					return err
				}
				return s.announceEncrypted(w(l)(bodyProducer))
			}
		}
	}
}

func Teardown(l logrus.FieldLogger) func() {
	return func() {
		ctx, span := otel.GetTracerProvider().Tracer("atlas-login").Start(context.Background(), "teardown")
		defer span.End()

		_ = tenant.ForAll(func(t tenant.Model) error {
			p := NewProcessor(l, tenant.WithContext(ctx, t))
			return model.ForEachSlice(p.AllInTenantProvider, p.Destroy)
		})
	}
}

func SendPing(l logrus.FieldLogger, t tenant.Model, wp writer.Producer) socket.IdleNotifier {
	return func(sessionId uuid.UUID) {
		s, ok := getRegistry().Get(t.Id(), sessionId)
		if !ok {
			return
		}
		l.Debugf("Session [%s] idle, sending PING.", sessionId)
		_ = Announce(l)(wp)(writer.Ping)(s, writer.PingBody())
	}
}
