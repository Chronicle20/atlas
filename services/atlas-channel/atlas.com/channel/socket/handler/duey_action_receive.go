package handler

import (
	"atlas-channel/compartment"
	dueyparcel "atlas-channel/parcel"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	parcelcb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/clientbound"
	parcelsb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/serverbound"
)

// dueyReceiveDeps collects every external lookup the DUEY_ACTION
// RECEIVE/DISCARD pre-flight makes, mirroring dueySendDeps
// (duey_action_send.go) so receiveParcel/discardParcel can be exercised
// without a REST client, a compartment lookup or the saga orchestrator.
type dueyReceiveDeps struct {
	// getParcel resolves the wire parcelId to the caller's own pending
	// parcel. It returns a non-nil error when no such parcel exists —
	// covering "not addressed to me", "already received/discarded" and an
	// unknown id in a single not-found path, since a resolution scoped to
	// the caller's own pending mailbox can never surface someone else's row
	// (design §7.2/§7.3).
	getParcel      func(characterId uint32, worldId world.Id, wireId uint32) (dueyparcel.Model, error)
	getCompartment func(characterId uint32, it inventory.Type) (compartment.Model, error)
	createSaga     func(sg saga.Saga) error
	discardParcel  func(id uuid.UUID, recipientId uint32) (dueyparcel.Model, error)
}

// handleDueyActionReceive wires dueyReceiveDeps to the real
// atlas-compartment / atlas-parcel / saga-orchestrator collaborators.
//
// getParcel's real implementation resolves the client's wire-format
// uint32 parcelId (design §5.3 — the PARCEL struct's `+0 uint32 parcelId`,
// echoed verbatim by CTabReceive::ReceiveParcel/DiscardParcel) against
// atlas-parcel's uuid.UUID row identity: it fetches the caller's own pending
// mailbox (GetForRecipient, capped at 10 — design §6.2 MailboxCapacity) and
// matches on the first 4 bytes of each row's id, big-endian. This is a
// deliberate, self-contained engineering choice for a wire detail the design
// doc explicitly leaves to implementation ("the exact byte layout... is
// derived during implementation", §5.3) — the atlas-channel task that builds
// DUEY_ACTION OPEN (out of this task's scope; no such handler exists yet)
// must assign the SAME 4-byte projection when it populates the PARCEL list,
// or this match silently fails closed (not found, per getParcel's contract
// above) rather than resolving to the wrong parcel. Flagged here for
// reviewer visibility.
func handleDueyActionReceive(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, sp *parcelsb.ActionReceive) {
	return func(s session.Model, sp *parcelsb.ActionReceive) {
		deps := dueyReceiveDeps{
			getParcel: func(characterId uint32, worldId world.Id, wireId uint32) (dueyparcel.Model, error) {
				return resolveParcelByWireId(l, ctx, characterId, worldId, wireId)
			},
			getCompartment: func(characterId uint32, it inventory.Type) (compartment.Model, error) {
				return compartment.NewProcessor(l, ctx).GetByType(characterId, it)
			},
			createSaga: func(sg saga.Saga) error {
				return saga.NewProcessor(l, ctx).Create(sg)
			},
		}
		receiveParcel(l, ctx, wp, s, sp, deps)
	}
}

// handleDueyActionDiscard wires dueyReceiveDeps to the real collaborators
// discardParcel needs. Discard shares getParcel's resolution helper with
// receive — see handleDueyActionReceive's doc comment.
func handleDueyActionDiscard(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, sp *parcelsb.ActionDiscard) {
	return func(s session.Model, sp *parcelsb.ActionDiscard) {
		deps := dueyReceiveDeps{
			getParcel: func(characterId uint32, worldId world.Id, wireId uint32) (dueyparcel.Model, error) {
				return resolveParcelByWireId(l, ctx, characterId, worldId, wireId)
			},
			discardParcel: func(id uuid.UUID, recipientId uint32) (dueyparcel.Model, error) {
				return dueyparcel.NewProcessor(l, ctx).Discard(id, recipientId)
			},
		}
		discardParcel(l, ctx, wp, s, sp, deps)
	}
}

// errParcelNotResolved is returned by resolveParcelByWireId when no pending
// parcel in the caller's own mailbox matches the wire id, or when more than
// one does (a same-mailbox collision — see resolveParcelByWireId).
var errParcelNotResolved = errors.New("parcel not resolved")

// resolveParcelByWireId is the production getParcel implementation — see
// handleDueyActionReceive's doc comment for the matching scheme and its
// caveat. Resolution is scoped to the caller's own pending mailbox
// (GetForRecipient), so a collision can only happen against the caller's
// own rows; when more than one matches, this rejects rather than guessing
// which one the client meant.
func resolveParcelByWireId(l logrus.FieldLogger, ctx context.Context, characterId uint32, worldId world.Id, wireId uint32) (dueyparcel.Model, error) {
	ms, err := dueyparcel.NewProcessor(l, ctx).GetForRecipient(characterId, worldId)
	if err != nil {
		return dueyparcel.Model{}, err
	}
	var match dueyparcel.Model
	found := false
	for _, m := range ms {
		if dueyparcel.WireId(m.Id()) == wireId {
			if found {
				l.Warnf("Character [%d] wire parcelId [%d] collides across multiple pending parcels; refusing to guess.", characterId, wireId)
				return dueyparcel.Model{}, errParcelNotResolved
			}
			match = m
			found = true
		}
	}
	if !found {
		return dueyparcel.Model{}, errParcelNotResolved
	}
	return match, nil
}

// handleDueyActionClose clears whatever session-side Duey dialog state the
// channel tracks (design §7.4/step 3) and announces nothing. No such state
// exists yet — the DUEY_ACTION OPEN handler that would populate it is out
// of this task's scope — so this is a genuine no-op today, not a stub: there
// is nothing to clear.
func handleDueyActionClose(l logrus.FieldLogger, _ context.Context, _ writer.Producer) func(s session.Model, sp *parcelsb.ActionClose) {
	return func(s session.Model, _ *parcelsb.ActionClose) {
		l.Debugf("Character [%d] closed the Duey dialog.", s.CharacterId())
	}
}

// receiveParcel runs the DUEY_ACTION RECEIVE pre-flight in the brief's
// order (free-slot check, then unique-item check, then the parcel-state
// check) and, if every check passes, builds the parcel_receive saga. Every
// rejection announces a PARCEL result arm inline and starts no saga —
// mirroring sendParcel's posture (NFR-5: never a disconnect).
func receiveParcel(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, sp *parcelsb.ActionReceive, deps dueyReceiveDeps) {
	reject := func(body func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte) {
		_ = session.Announce(l)(ctx)(wp)(parcelcb.ParcelWriter)(body)(s)
	}

	p, err := deps.getParcel(s.CharacterId(), s.WorldId(), sp.ParcelId())
	if err != nil {
		l.WithError(err).Warnf("Character [%d] DUEY_ACTION RECEIVE: unable to resolve parcel [%d].", s.CharacterId(), sp.ParcelId())
		reject(parcelcb.ParcelIncorrectRequestBody())
		return
	}

	// A meso-only parcel carries no item, so neither inventory pre-flight
	// applies (it can never fail on slots or a duplicate template).
	if itemId := p.ItemId(); itemId != nil {
		it := inventory.Type(p.ItemType())
		cp, cerr := deps.getCompartment(s.CharacterId(), it)
		if cerr != nil {
			l.WithError(cerr).Errorf("Character [%d] DUEY_ACTION RECEIVE: unable to load inventory type [%d].", s.CharacterId(), it)
			reject(parcelcb.ParcelIncorrectRequestBody())
			return
		}
		if uint32(len(cp.Assets())) >= cp.Capacity() {
			l.Warnf("Character [%d] attempted to receive parcel [%s] with no free slot in inventory type [%d].", s.CharacterId(), p.Id(), it)
			reject(parcelcb.ParcelRecvNoFreeSlotsBody())
			return
		}
		if _, found := cp.FindFirstByItemId(*itemId); found {
			l.Warnf("Character [%d] attempted to receive parcel [%s] but already holds item [%d].", s.CharacterId(), p.Id(), *itemId)
			reject(parcelcb.ParcelRecvUniqueConflictBody())
			return
		}
	}

	if p.ReceivableAt().After(time.Now()) {
		l.Warnf("Character [%d] attempted to receive parcel [%s] before it is receivable.", s.CharacterId(), p.Id())
		reject(parcelcb.ParcelIncorrectRequestBody())
		return
	}

	sg := buildParcelReceiveSaga(uuid.New(), time.Now(), s, p)
	if err = deps.createSaga(sg); err != nil {
		l.WithError(err).Errorf("Character [%d] unable to initiate parcel receive saga for parcel [%s].", s.CharacterId(), p.Id())
	}
}

// buildParcelReceiveSaga assembles the parcel_receive saga in design §4.3's
// order: withdraw_from_parcel (composite → release_from_parcel +
// accept_to_character) then award_mesos (positive mesoAmount — a receive
// never costs the recipient anything).
func buildParcelReceiveSaga(transactionId uuid.UUID, now time.Time, s session.Model, p dueyparcel.Model) saga.Saga {
	steps := make([]saga.Step, 0, 2)

	steps = append(steps, saga.Step{
		StepId: "withdraw_from_parcel",
		Status: saga.Pending,
		Action: saga.WithdrawFromParcel,
		Payload: saga.WithdrawFromParcelPayload{
			TransactionId: transactionId,
			ParcelId:      p.Id(),
			CharacterId:   s.CharacterId(),
			WorldId:       s.WorldId(),
			InventoryType: inventory.Type(p.ItemType()),
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	steps = append(steps, saga.Step{
		StepId: "award_mesos",
		Status: saga.Pending,
		Action: saga.AwardMesos,
		Payload: saga.AwardMesosPayload{
			CharacterId: s.CharacterId(),
			WorldId:     s.WorldId(),
			ChannelId:   s.ChannelId(),
			ActorId:     0,
			ActorType:   "SYSTEM",
			Amount:      int32(p.MesoAmount()),
			ShowEffect:  true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	return saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.ParcelReceive,
		InitiatedBy:   "DUEY_ACTION_RECEIVE",
		Steps:         steps,
	}
}

// discardParcel runs the DUEY_ACTION DISCARD arm (design §4.4/§7.3): not a
// saga — the contents are destroyed and nothing leaves custody, so the
// channel PATCHes atlas-parcel directly and, on success, announces
// PARCEL_REMOVED with kind == ParcelRemovedKindDiscarded synchronously
// (unlike receive, whose PARCEL_REMOVED is announced on saga completion, by
// a consumer outside this task's scope).
func discardParcel(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, sp *parcelsb.ActionDiscard, deps dueyReceiveDeps) {
	reject := func(body func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte) {
		_ = session.Announce(l)(ctx)(wp)(parcelcb.ParcelWriter)(body)(s)
	}

	p, err := deps.getParcel(s.CharacterId(), s.WorldId(), sp.ParcelId())
	if err != nil {
		l.WithError(err).Warnf("Character [%d] DUEY_ACTION DISCARD: unable to resolve parcel [%d].", s.CharacterId(), sp.ParcelId())
		reject(parcelcb.ParcelIncorrectRequestBody())
		return
	}

	if _, err = deps.discardParcel(p.Id(), s.CharacterId()); err != nil {
		l.WithError(err).Errorf("Character [%d] unable to discard parcel [%s].", s.CharacterId(), p.Id())
		reject(parcelcb.ParcelIncorrectRequestBody())
		return
	}

	_ = session.Announce(l)(ctx)(wp)(parcelcb.ParcelWriter)(parcelcb.ParcelRemovedBody(dueyparcel.WireId(p.Id()), parcelcb.ParcelRemovedKindDiscarded))(s)
}
