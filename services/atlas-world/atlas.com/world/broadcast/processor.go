package broadcast

import (
	"context"
	"time"

	kmessage "atlas-world/kafka/message"
	bmessage "atlas-world/kafka/message/broadcast"
	bproducer "atlas-world/kafka/producer/broadcast"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor defines broadcast queue operations: enqueueing a new entry
// (megaphone or Maple TV) and sweeping expired active entries for a tenant.
type Processor interface {
	// Enqueue appends e to the (worldId, family) queue. If the queue was
	// idle, e activates immediately (STARTED emitted) in addition to the
	// QUEUED event (waitSeconds 0). If something was already active/pending,
	// only QUEUED is emitted, carrying the wait computed from the queue as
	// it stood immediately before the append.
	Enqueue(worldId world.Id, family string, e Entry) error
	// GetQueue returns the current QueueModel for (worldId, family). Returns
	// atlas-redis's ErrNotFound (unwrapped) if no queue has been created yet
	// for this tenant/world/family; callers that want "no queue" to read as
	// an empty queue must translate that themselves.
	GetQueue(worldId world.Id, family string) (QueueModel, error)
	// SweepTenant expires and promotes queues for the tenant bound to this
	// processor's context. Intended to be called once per tick, per tenant,
	// by the leader-elected sweep task (Task 9).
	SweepTenant() error
}

// ProcessorImpl implements Processor.
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

// NewProcessor creates a new broadcast processor scoped to the tenant found
// in ctx.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetQueue(worldId world.Id, family string) (QueueModel, error) {
	return GetRegistry().Get(p.ctx, p.t, worldId, family)
}

// Enqueue appends e to the (worldId, family) queue under CAS and emits the
// resulting status event(s). The CAS `fn` closures below are pure: they only
// compute a new QueueModel and stash observations (the pre-append wait, the
// newly-activated entry) into local variables for the *caller* to read after
// Upsert returns. Because TenantRegistry.Update retries on WATCH/EXEC
// contention, `fn` may run more than once per Upsert call — each retry
// simply overwrites the stashed variables with its own attempt's values, so
// by the time Upsert returns (successfully, exactly once), the stashed
// values reflect precisely the winning attempt. No emission happens inside
// `fn` itself; emission happens only after Upsert returns, using the
// winning attempt's observations. This satisfies design D1 (only the CAS
// winner emits) and keeps `fn` side-effect free as required.
//
// The append and the "was this queue idle, so activate immediately" check
// are done inside ONE Upsert/fn — NOT two separate Upsert calls. This is
// load-bearing, not stylistic: ActivateNext unconditionally promotes the
// Pending head into Active (it does not check whether Active is already
// set), so if the idle-check and the activation were two separate CAS
// transactions, two concurrent Enqueue calls on the same idle queue could
// both observe Active==nil after their own append (since neither has
// activated yet) and both then call ActivateNext in a second transaction —
// the second one would silently clobber the first one's freshly-activated
// entry, losing it. Doing both steps inside the same fn means the "was it
// idle" check and the activation are evaluated against the exact same
// `current` snapshot, atomically, so only one concurrent caller's append
// can ever be the one that also activates.
func (p *ProcessorImpl) Enqueue(worldId world.Id, family string, e Entry) error {
	var preAppendWaitSeconds uint32
	var activated *Entry

	_, err := GetRegistry().Upsert(p.ctx, p.t, worldId, family, func(current QueueModel) QueueModel {
		now := time.Now()
		preAppendWaitSeconds = current.WaitSeconds(now)
		activated = nil

		appended := current.Append(e)
		if appended.Active != nil {
			// Something was already active or pending before this append;
			// the entry waits behind it. No activation happens here.
			return appended
		}

		// Queue was idle: activate the entry we just appended (now the head
		// of Pending) in this same transaction.
		next, act := appended.ActivateNext(now)
		activated = act
		return next
	})
	if err != nil {
		return err
	}

	l := p.l.WithFields(logrus.Fields{
		"tenant":      p.t.Id().String(),
		"worldId":     worldId,
		"family":      family,
		"characterId": e.CharacterId,
	})

	return kmessage.Emit(kproducer.ProviderImpl(p.l)(p.ctx))(func(mb *kmessage.Buffer) error {
		if activated == nil {
			// Something was already active or pending before this append;
			// the entry waits behind it.
			if err := mb.Put(bmessage.EnvEventTopicWorldBroadcastStatus, bproducer.QueuedStatusEventProvider(worldId, family, e.CharacterId, preAppendWaitSeconds)); err != nil {
				return err
			}
			l.WithField("waitSeconds", preAppendWaitSeconds).Info("Enqueued broadcast entry.")
			return nil
		}

		if err := mb.Put(bmessage.EnvEventTopicWorldBroadcastStatus, bproducer.StartedStatusEventProvider(worldId, family, startedPayload(*activated))); err != nil {
			return err
		}
		l.Info("Activated broadcast entry immediately (queue was idle).")

		if err := mb.Put(bmessage.EnvEventTopicWorldBroadcastStatus, bproducer.QueuedStatusEventProvider(worldId, family, e.CharacterId, 0)); err != nil {
			return err
		}
		l.WithField("waitSeconds", uint32(0)).Info("Enqueued broadcast entry.")
		return nil
	})
}

// SweepTenant expires active entries whose deadline has passed and promotes
// their queue's next pending entry, for every (worldId, family) queue
// belonging to the tenant bound to this processor's context.
func (p *ProcessorImpl) SweepTenant() error {
	now := time.Now()

	queues, err := GetRegistry().AllQueues(p.ctx, p.t)
	if err != nil {
		return err
	}

	return kmessage.Emit(kproducer.ProviderImpl(p.l)(p.ctx))(func(mb *kmessage.Buffer) error {
		for key, snapshot := range queues {
			if !snapshot.ActiveExpired(now) {
				continue
			}
			if err := p.sweepQueue(mb, now, key.WorldId, key.Family); err != nil {
				return err
			}
		}
		return nil
	})
}

// sweepQueue re-checks expiry under CAS (the snapshot driving the caller's
// loop may be stale) and, if still expired, clears the active entry and
// promotes the next pending entry (if any), emitting ENDED and, when a next
// entry activated, STARTED. Like Enqueue's closures, the CAS `fn` here is
// pure: it only stashes the expired/activated entries into local variables,
// which the caller reads only after Upsert has returned successfully.
func (p *ProcessorImpl) sweepQueue(mb *kmessage.Buffer, now time.Time, worldId world.Id, family string) error {
	var expired *Entry
	var activated *Entry

	if _, err := GetRegistry().Upsert(p.ctx, p.t, worldId, family, func(current QueueModel) QueueModel {
		expired = nil
		activated = nil
		if !current.ActiveExpired(now) {
			return current
		}
		expired = current.Active
		next, act := current.ClearActive().ActivateNext(now)
		activated = act
		return next
	}); err != nil {
		return err
	}

	if expired == nil {
		// Another sweep (or nothing) already handled this queue between our
		// snapshot read and this CAS attempt; nothing to emit.
		return nil
	}

	l := p.l.WithFields(logrus.Fields{
		"tenant":      p.t.Id().String(),
		"worldId":     worldId,
		"family":      family,
		"characterId": expired.CharacterId,
	})
	if err := mb.Put(bmessage.EnvEventTopicWorldBroadcastStatus, bproducer.EndedStatusEventProvider(worldId, family, expired.CharacterId)); err != nil {
		return err
	}
	l.Info("Ended expired broadcast entry.")

	if activated != nil {
		if err := mb.Put(bmessage.EnvEventTopicWorldBroadcastStatus, bproducer.StartedStatusEventProvider(worldId, family, startedPayload(*activated))); err != nil {
			return err
		}
		p.l.WithFields(logrus.Fields{
			"tenant":      p.t.Id().String(),
			"worldId":     worldId,
			"family":      family,
			"characterId": activated.CharacterId,
		}).Info("Activated next broadcast entry.")
	}

	return nil
}

// startedPayload maps an activated domain Entry onto the message-package
// StartedPayload the producer package accepts, keeping the domain type out
// of the producer package's import graph (see StartedPayload's doc comment).
func startedPayload(e Entry) bmessage.StartedPayload {
	return bmessage.StartedPayload{
		CharacterId:     e.CharacterId,
		DurationSeconds: e.DurationSeconds,
		ChannelId:       e.Payload.ChannelId,
		SenderName:      e.Payload.SenderName,
		SenderMedal:     e.Payload.SenderMedal,
		Messages:        e.Payload.Messages,
		WhispersOn:      e.Payload.WhispersOn,
		ItemId:          e.Payload.ItemId,
		TvMessageType:   e.Payload.TvMessageType,
		SenderLook:      e.Payload.SenderLook,
		ReceiverName:    e.Payload.ReceiverName,
		ReceiverLook:    e.Payload.ReceiverLook,
	}
}
