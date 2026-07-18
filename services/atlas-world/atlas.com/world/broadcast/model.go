package broadcast

import (
	"fmt"
	"math"
	"time"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/google/uuid"
)

// Family identifies which serialized world-broadcast queue an entry belongs
// to: TV (Maple TV) or AVATAR (avatar megaphone).
const (
	FamilyTV     = "TV"
	FamilyAvatar = "AVATAR"
)

// Payload is the render payload carried from enqueue to STARTED, verbatim.
type Payload struct {
	ChannelId     byte                       `json:"channelId"`
	SenderName    string                     `json:"senderName"`
	SenderMedal   string                     `json:"senderMedal"`
	Messages      []string                   `json:"messages"`
	WhispersOn    bool                       `json:"whispersOn"`
	ItemId        uint32                     `json:"itemId"`
	TvMessageType string                     `json:"tvMessageType"` // semantic key NORMAL|STAR|HEART; resolved to a client wire byte via the tenant messageTypes writer table at the packet layer, never carried as a byte here (DOM-25)
	SenderLook    sharedsaga.AvatarSnapshot  `json:"senderLook"`
	ReceiverName  string                     `json:"receiverName"`
	ReceiverLook  *sharedsaga.AvatarSnapshot `json:"receiverLook,omitempty"`
}

// Entry is one queued (or active) broadcast request.
type Entry struct {
	Id              uuid.UUID `json:"id"`
	CharacterId     uint32    `json:"characterId"`
	Payload         Payload   `json:"payload"`
	DurationSeconds uint32    `json:"durationSeconds"`
	ActivatedAt     time.Time `json:"activatedAt,omitempty"`
	ExpiresAt       time.Time `json:"expiresAt,omitempty"`
}

// NewEntry constructs a validated Entry for enqueueing into the given
// broadcast family. family is not stored on Entry itself (it selects which
// (worldId, family) queue the entry is appended to - see
// Processor.Enqueue) but is validated here so that no caller can build an
// Entry destined for an unrecognized family via a raw struct literal.
// ActivatedAt/ExpiresAt are intentionally not accepted here: they are only
// ever stamped by QueueModel.ActivateNext, never at construction time.
func NewEntry(family string, id uuid.UUID, characterId uint32, payload Payload, durationSeconds uint32) (Entry, error) {
	if family != FamilyTV && family != FamilyAvatar {
		return Entry{}, fmt.Errorf("unhandled broadcast family [%s]", family)
	}
	return Entry{
		Id:              id,
		CharacterId:     characterId,
		Payload:         payload,
		DurationSeconds: durationSeconds,
	}, nil
}

// QueueModel is the per (tenant, world, family) queue stored as one Redis
// JSON value and mutated only through TenantRegistry.Update (WATCH/CAS).
type QueueModel struct {
	Active  *Entry  `json:"active,omitempty"`
	Pending []Entry `json:"pending"`
}

// Append adds e to the tail of Pending, preserving existing order. Pure:
// returns a modified copy, no I/O.
func (q QueueModel) Append(e Entry) QueueModel {
	pending := make([]Entry, 0, len(q.Pending)+1)
	pending = append(pending, q.Pending...)
	pending = append(pending, e)
	return QueueModel{
		Active:  q.Active,
		Pending: pending,
	}
}

// ActivateNext pops the head of Pending into Active, stamping
// ActivatedAt=now and ExpiresAt=now+DurationSeconds. Returns the modified
// queue and a pointer to the newly activated entry, or nil if Pending was
// empty. Pure: no I/O, now is passed in.
func (q QueueModel) ActivateNext(now time.Time) (QueueModel, *Entry) {
	if len(q.Pending) == 0 {
		return q, nil
	}

	head := q.Pending[0]
	head.ActivatedAt = now
	head.ExpiresAt = now.Add(time.Duration(head.DurationSeconds) * time.Second)

	pending := make([]Entry, len(q.Pending)-1)
	copy(pending, q.Pending[1:])

	return QueueModel{
		Active:  &head,
		Pending: pending,
	}, &head
}

// ClearActive removes the Active entry, leaving Pending untouched. Pure.
func (q QueueModel) ClearActive() QueueModel {
	return QueueModel{
		Active:  nil,
		Pending: q.Pending,
	}
}

// ActiveExpired reports whether the Active entry (if any) has expired at
// now. The boundary is inclusive: now == ExpiresAt is expired.
func (q QueueModel) ActiveExpired(now time.Time) bool {
	if q.Active == nil {
		return false
	}
	return !now.Before(q.Active.ExpiresAt)
}

// WaitSeconds returns the estimated wait, in seconds, before a
// newly-enqueued entry would activate: the remaining time on the Active
// entry (rounded up to the next whole second) plus the sum of every
// Pending entry's DurationSeconds. Returns 0 when the queue is idle
// (no Active entry, no Pending entries).
func (q QueueModel) WaitSeconds(now time.Time) uint32 {
	var total uint32

	if q.Active != nil {
		remaining := q.Active.ExpiresAt.Sub(now)
		if remaining > 0 {
			total += uint32(math.Ceil(remaining.Seconds()))
		}
	}

	for _, e := range q.Pending {
		total += e.DurationSeconds
	}

	return total
}
