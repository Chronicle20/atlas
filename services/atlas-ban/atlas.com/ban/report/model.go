package report

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidStatus = errors.New("invalid report status")
	ErrInvalidKind   = errors.New("invalid report kind")
)

// Server-side caps on client-supplied strings (NFR): truncate-and-log,
// never reject — a truncated report is more useful to a GM than a vanished
// one. Description is capped in characters (bytes), chat log in bytes.
const (
	MaxDescriptionLength = 2000
	MaxChatLogBytes      = 16384
)

// Claim quota. A reporter may create at most MaxClaimsPerWindow claims whose
// created_at falls inside a rolling ClaimQuotaWindow ending now — there is no
// reset boundary, so the oldest claim ages out of the count as time passes.
// The client's own copy calls this a weekly allowance (StringPool SP_3381
// "you have %d reports left this week"), which the rolling window approximates
// without giving players a burst at a fixed reset.
//
// Sue reports are deliberately outside this cap: the client's sue result has
// its own daily-limit code (SP_3005, "you may only report users 10 times a
// day") that Atlas does not enforce.
const (
	MaxClaimsPerWindow = 100
	ClaimQuotaWindow   = 7 * 24 * time.Hour
)

type Kind string

const (
	KindSue   Kind = "sue"
	KindClaim Kind = "claim"
)

func (k Kind) Valid() bool {
	return k == KindSue || k == KindClaim
}

type Status string

const (
	StatusOpen     Status = "open"
	StatusReviewed Status = "reviewed"
	StatusActioned Status = "actioned"
)

func (s Status) Valid() bool {
	return s == StatusOpen || s == StatusReviewed || s == StatusActioned
}

// TranscriptLine is one server-captured chat line attached to a report.
type TranscriptLine struct {
	Timestamp  int64  `json:"timestamp"`
	SenderId   uint32 `json:"senderId"`
	SenderName string `json:"senderName"`
	ChatType   string `json:"chatType"`
	Text       string `json:"text"`
}

type Model struct {
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

func (m Model) Id() uuid.UUID                      { return m.id }
func (m Model) TenantId() uuid.UUID                { return m.tenantId }
func (m Model) Kind() Kind                         { return m.kind }
func (m Model) ReporterId() uint32                 { return m.reporterId }
func (m Model) ReporterName() string               { return m.reporterName }
func (m Model) AccusedId() uint32                  { return m.accusedId }
func (m Model) AccusedName() string                { return m.accusedName }
func (m Model) ReasonType() byte                   { return m.reasonType }
func (m Model) Description() string                { return m.description }
func (m Model) ChatLog() *string                   { return m.chatLog }
func (m Model) ServerTranscript() []TranscriptLine { return m.serverTranscript }
func (m Model) Status() Status                     { return m.status }
func (m Model) CreatedAt() time.Time               { return m.createdAt }
func (m Model) UpdatedAt() time.Time               { return m.updatedAt }
