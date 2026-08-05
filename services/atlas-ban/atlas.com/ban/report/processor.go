package report

import (
	"atlas-ban/character"
	"atlas-ban/chat"
	"atlas-ban/kafka/message"
	report2 "atlas-ban/kafka/message/report"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	CreateFromCommand(buf *message.Buffer) func(c report2.CreateCommandBody) error
	CreateFromCommandAndEmit(c report2.CreateCommandBody) error
	UpdateStatus(reportId uuid.UUID, status Status) (Model, error)
	GetById(reportId uuid.UUID) (Model, error)
	ByIdProvider(reportId uuid.UUID) model.Provider[Model]
	GetByTenant() ([]Model, error)
	GetByStatus(status Status) ([]Model, error)
}

type ProcessorImpl struct {
	l     logrus.FieldLogger
	ctx   context.Context
	db    *gorm.DB
	t     tenant.Model
	p     producer.Provider
	charP character.Processor
	chatP chat.Processor
}

var _ Processor = (*ProcessorImpl)(nil)

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return NewProcessorWithClients(l, ctx, db, character.NewProcessor(l, ctx), chat.NewProcessor(l, ctx))
}

// NewProcessorWithClients constructs a Processor with explicit REST client
// implementations. Production callers use NewProcessor; callers that already
// hold client instances (or substitutes, e.g. tests) inject them here.
func NewProcessorWithClients(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, charP character.Processor, chatP chat.Processor) Processor {
	return &ProcessorImpl{
		l:     l,
		ctx:   ctx,
		db:    db,
		t:     tenant.MustFromContext(ctx),
		p:     producer.ProviderImpl(l)(ctx),
		charP: charP,
		chatP: chatP,
	}
}

func (p *ProcessorImpl) CreateFromCommandAndEmit(c report2.CreateCommandBody) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.CreateFromCommand(buf)(c)
	})
}

// CreateFromCommand resolves the accused, snapshots the chat transcript,
// persists the report, and buffers exactly one status event (CREATED or
// ERROR). Business rejections (unresolvable accused, DB failure) return nil
// so the ERROR event still emits via message.Emit — that function skips
// emission entirely when the callback errors, so a genuine rejection must
// buffer its failure event and return nil, not an error, or the reporter
// never gets a result packet.
func (p *ProcessorImpl) CreateFromCommand(buf *message.Buffer) func(c report2.CreateCommandBody) error {
	return func(c report2.CreateCommandBody) error {
		fail := func(code string) error {
			return buf.Put(report2.EnvEventTopicStatus, statusEventProvider(uuid.Nil, Kind(c.Kind), c.WorldId, c.ReporterId, report2.EventStatusError, code))
		}

		reporter, err := p.charP.GetById(c.ReporterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to resolve reporter [%d] for report.", c.ReporterId)
			return fail(report2.ErrorCodeInternal)
		}

		var accused character.Model
		switch {
		case c.AccusedName != "":
			accused, err = p.charP.GetByName(c.AccusedName)
		case c.AccusedId != 0:
			accused, err = p.charP.GetById(c.AccusedId)
		default:
			err = model.ErrNoResultFound
		}
		if err != nil {
			// FirstProvider (libs/atlas-model/model/processor.go:552) returns
			// ErrEmptySlice when the by-name provider yields zero characters —
			// that's the path GetByName's zero-filter First() call actually
			// takes for an unknown name. ErrNoResultFound (line 571) is what
			// First() returns when filters were supplied and none matched;
			// GetByName supplies none today, so that branch is unreachable
			// here, but it's mapped too for defensiveness against a future
			// filtered GetByName. requests.ErrNotFound covers GetById's 404.
			if errors.Is(err, requests.ErrNotFound) || errors.Is(err, model.ErrEmptySlice) || errors.Is(err, model.ErrNoResultFound) {
				p.l.Infof("Rejecting [%s] report from [%d]: accused [%s/%d] not found in tenant.", c.Kind, c.ReporterId, c.AccusedName, c.AccusedId)
				return fail(report2.ErrorCodeNotFound)
			}
			p.l.WithError(err).Errorf("Unable to resolve accused [%s/%d] for report from [%d].", c.AccusedName, c.AccusedId, c.ReporterId)
			return fail(report2.ErrorCodeInternal)
		}

		description := c.Description
		if len(description) > MaxDescriptionLength {
			p.l.Warnf("Truncating report description from [%d] to [%d] chars for reporter [%d].", len(description), MaxDescriptionLength, c.ReporterId)
			description = description[:MaxDescriptionLength]
		}
		var chatLog *string
		if c.ChatClaim {
			cl := c.ChatLog
			if len(cl) > MaxChatLogBytes {
				p.l.Warnf("Truncating report chat log from [%d] to [%d] bytes for reporter [%d].", len(cl), MaxChatLogBytes, c.ReporterId)
				cl = cl[:MaxChatLogBytes]
			}
			chatLog = &cl
		}

		// Transcript is corroboration, best-effort by design: a messages
		// outage degrades to a null transcript, never a failed report.
		var transcript []TranscriptLine
		lines, terr := p.chatP.RecentInvolving([]uint32{c.ReporterId, accused.Id()})
		if terr != nil {
			p.l.WithError(terr).Warnf("Unable to fetch chat transcript for report from [%d]; persisting without.", c.ReporterId)
		} else {
			transcript = make([]TranscriptLine, 0, len(lines))
			for _, line := range lines {
				transcript = append(transcript, TranscriptLine{
					Timestamp:  line.Timestamp(),
					SenderId:   line.SenderId(),
					SenderName: line.SenderName(),
					ChatType:   line.ChatType(),
					Text:       line.Text(),
				})
			}
		}

		m, err := create(p.db.WithContext(p.ctx))(p.t.Id(), Kind(c.Kind), c.ReporterId, reporter.Name(), accused.Id(), accused.Name(), c.ReasonType, description, chatLog, transcript)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to persist [%s] report from [%d].", c.Kind, c.ReporterId)
			return fail(report2.ErrorCodeInternal)
		}
		p.l.Infof("Created [%s] report [%s]: reporter [%d/%s] accused [%d/%s] reason [%d].", m.Kind(), m.Id(), m.ReporterId(), m.ReporterName(), m.AccusedId(), m.AccusedName(), m.ReasonType())
		return buf.Put(report2.EnvEventTopicStatus, statusEventProvider(m.Id(), m.Kind(), c.WorldId, c.ReporterId, report2.EventStatusCreated, ""))
	}
}

func (p *ProcessorImpl) UpdateStatus(reportId uuid.UUID, status Status) (Model, error) {
	if !status.Valid() {
		return Model{}, ErrInvalidStatus
	}
	err := updateStatus(p.db.WithContext(p.ctx))(reportId, status)
	if err != nil {
		return Model{}, err
	}
	m, err := p.GetById(reportId)
	if err != nil {
		return Model{}, err
	}
	p.l.Infof("Report [%s] status updated to [%s].", reportId, status)
	return m, nil
}

func (p *ProcessorImpl) GetById(reportId uuid.UUID) (Model, error) {
	return p.ByIdProvider(reportId)()
}

func (p *ProcessorImpl) ByIdProvider(reportId uuid.UUID) model.Provider[Model] {
	return model.Map(Make)(entityById(reportId)(p.db.WithContext(p.ctx)))
}

func (p *ProcessorImpl) GetByTenant() ([]Model, error) {
	return model.SliceMap(Make)(entitiesByTenant()(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetByStatus(status Status) ([]Model, error) {
	return model.SliceMap(Make)(entitiesByStatus(status)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}
