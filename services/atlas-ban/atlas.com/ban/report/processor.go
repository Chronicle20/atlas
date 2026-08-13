package report

import (
	"atlas-ban/character"
	"atlas-ban/chat"
	"atlas-ban/kafka/message"
	report2 "atlas-ban/kafka/message/report"
	"context"
	"errors"
	"time"
	"unicode/utf8"

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

		// Quota is checked before anything else the create path does: it needs
		// only the reporter id, it is the cheapest rejection available, and
		// rejecting here means the channel never charges the claim fee for a
		// report that was never going to be stored.
		var remaining int32
		if Kind(c.Kind) == KindClaim {
			used, cerr := countClaimsByReporterSince(p.db.WithContext(p.ctx))(c.ReporterId, time.Now().Add(-ClaimQuotaWindow))
			if cerr != nil {
				p.l.WithError(cerr).Errorf("Unable to count recent claims for reporter [%d].", c.ReporterId)
				return fail(report2.ErrorCodeInternal)
			}
			if used >= MaxClaimsPerWindow {
				p.l.Infof("Rejecting claim from [%d]: [%d] claims in the last [%s] meets the cap of [%d].", c.ReporterId, used, ClaimQuotaWindow, MaxClaimsPerWindow)
				return fail(report2.ErrorCodeQuotaExceeded)
			}
			remaining = int32(MaxClaimsPerWindow - used - 1)
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

		// Description's cap is in RUNES (characters), not bytes: byte-slicing
		// would keep fewer than MaxDescriptionLength characters for any
		// non-ASCII input and can split a multi-byte rune, producing invalid
		// UTF-8 that Postgres rejects on INSERT — turning a truncate-and-log
		// into a create failure, the opposite of this cap's purpose.
		description := c.Description
		if runeCount := utf8.RuneCountInString(description); runeCount > MaxDescriptionLength {
			p.l.Warnf("Truncating report description from [%d] to [%d] chars for reporter [%d].", runeCount, MaxDescriptionLength, c.ReporterId)
			description = truncateRunes(description, MaxDescriptionLength)
		}
		var chatLog *string
		if c.ChatClaim {
			cl := c.ChatLog
			if len(cl) > MaxChatLogBytes {
				p.l.Warnf("Truncating report chat log from [%d] to [%d] bytes for reporter [%d].", len(cl), MaxChatLogBytes, c.ReporterId)
				// Cap stays byte-based (16384), but the cut must land on a
				// rune boundary for the same UTF-8-validity reason as above.
				cl = truncateBytesAtRuneBoundary(cl, MaxChatLogBytes)
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
		return buf.Put(report2.EnvEventTopicStatus, createdStatusEventProvider(m.Id(), m.Kind(), c.WorldId, c.ReporterId, m.Kind() == KindClaim, remaining))
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

// truncateRunes caps s at maxRunes RUNES (not bytes), so the result is
// always valid UTF-8 and never fewer than intended characters for
// non-ASCII input. A no-op when s already has maxRunes runes or fewer.
func truncateRunes(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	r := []rune(s)
	return string(r[:maxRunes])
}

// truncateBytesAtRuneBoundary caps s at maxBytes BYTES, walking rune-by-rune
// so the cut always lands on a full rune — never splitting a multi-byte
// sequence into an invalid one. A no-op when s already fits within maxBytes.
func truncateBytesAtRuneBoundary(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	end := 0
	for i := 0; i < len(s); {
		_, size := utf8.DecodeRuneInString(s[i:])
		if i+size > maxBytes {
			break
		}
		i += size
		end = i
	}
	return s[:end]
}
