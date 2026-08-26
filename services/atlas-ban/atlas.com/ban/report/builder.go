package report

import (
	"time"

	"github.com/google/uuid"
)

type Builder struct {
	id               uuid.UUID
	tenantId         uuid.UUID
	kind             Kind
	reporterId       uint32
	reporterName     string
	accusedId        uint32
	accusedName      string
	reasonType       byte
	description      string
	chatLog          *string
	serverTranscript []TranscriptLine
	status           Status
	createdAt        time.Time
	updatedAt        time.Time
}

func NewBuilder(tenantId uuid.UUID, kind Kind, reporterId uint32) *Builder {
	return &Builder{
		tenantId:   tenantId,
		kind:       kind,
		reporterId: reporterId,
		status:     StatusOpen,
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder            { b.id = id; return b }
func (b *Builder) SetReporterName(name string) *Builder   { b.reporterName = name; return b }
func (b *Builder) SetAccusedId(id uint32) *Builder        { b.accusedId = id; return b }
func (b *Builder) SetAccusedName(name string) *Builder    { b.accusedName = name; return b }
func (b *Builder) SetReasonType(reasonType byte) *Builder { b.reasonType = reasonType; return b }

func (b *Builder) SetDescription(description string) *Builder { b.description = description; return b }
func (b *Builder) SetChatLog(chatLog *string) *Builder        { b.chatLog = chatLog; return b }
func (b *Builder) SetServerTranscript(lines []TranscriptLine) *Builder {
	b.serverTranscript = lines
	return b
}
func (b *Builder) SetStatus(status Status) *Builder          { b.status = status; return b }
func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder { b.createdAt = createdAt; return b }
func (b *Builder) SetUpdatedAt(updatedAt time.Time) *Builder { b.updatedAt = updatedAt; return b }

func (b *Builder) Build() (Model, error) {
	if !b.kind.Valid() {
		return Model{}, ErrInvalidKind
	}
	if !b.status.Valid() {
		return Model{}, ErrInvalidStatus
	}
	return Model{
		id:               b.id,
		tenantId:         b.tenantId,
		kind:             b.kind,
		reporterId:       b.reporterId,
		reporterName:     b.reporterName,
		accusedId:        b.accusedId,
		accusedName:      b.accusedName,
		reasonType:       b.reasonType,
		description:      b.description,
		chatLog:          b.chatLog,
		serverTranscript: b.serverTranscript,
		status:           b.status,
		createdAt:        b.createdAt,
		updatedAt:        b.updatedAt,
	}, nil
}
