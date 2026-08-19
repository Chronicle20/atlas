package parcel

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

// ErrAlreadyReleased is returned by ReleaseCustody when the row was no
// longer pending at write time — a replayed RELEASE_FROM_PARCEL command
// (or a lost race against a concurrent release). It is not a failure: the
// caller re-acks success but must NOT re-emit the RELEASED status event,
// since the first delivery already emitted it.
var ErrAlreadyReleased = errors.New("parcel already released")

// AcceptParams carries every field the ACCEPT_TO_PARCEL command needs to
// create a pending parcel row from the command body alone — the wire fields
// of kafka/message/custody.AcceptToParcelCommandBody, mapped 1:1. HasItem
// false leaves the item fields unused and the row is created with a nil
// ItemId (meso-only parcel).
type AcceptParams struct {
	ParcelId           uuid.UUID
	CharacterId        uint32
	WorldId            world.Id
	SenderAccountId    uint32
	SenderName         string
	RecipientId        uint32
	RecipientAccountId uint32
	MesoAmount         uint32
	FeePaid            uint32
	Quick              bool
	Message            string
	ReceivableAt       time.Time
	ExpiresAt          time.Time

	HasItem bool

	// Item snapshot
	TemplateId    uint32
	Quantity      uint32
	Strength      uint16
	Dexterity     uint16
	Intelligence  uint16
	Luck          uint16
	HP            uint16
	MP            uint16
	WeaponAttack  uint16
	MagicAttack   uint16
	WeaponDefense uint16
	MagicDefense  uint16
	Accuracy      uint16
	Avoidability  uint16
	Hands         uint16
	Speed         uint16
	Jump          uint16
	Slots         uint16
	Level         byte
	ItemExp       uint32
	Flags         uint16
	Owner         string
}

// AcceptCustody creates the pending parcel row from the command body alone.
// ParcelId is allocated up-front by the initiator (the send saga), so
// creation is deterministic and idempotent on replay: a redelivery finds the
// row already present and returns it, still success (mirrors
// AcceptToMtsListing's idempotent create).
func (p *ProcessorImpl) AcceptCustody(params AcceptParams) (Model, error) {
	var result Model
	terr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		if existing, err := ById(params.ParcelId)(tx)(); err == nil {
			result = existing
			return nil
		}

		b := NewBuilder().
			SetId(params.ParcelId).
			SetWorldId(params.WorldId).
			SetSenderId(params.CharacterId).
			SetSenderAccountId(params.SenderAccountId).
			SetSenderName(params.SenderName).
			SetRecipientId(params.RecipientId).
			SetRecipientAccountId(params.RecipientAccountId).
			SetMessage(params.Message).
			SetMesoAmount(params.MesoAmount).
			SetFeePaid(params.FeePaid).
			SetStatus(StatusPending).
			SetQuick(params.Quick).
			SetCreatedAt(p.now()).
			SetReceivableAt(params.ReceivableAt).
			SetExpiresAt(params.ExpiresAt)

		if params.HasItem {
			templateId := params.TemplateId
			b.SetItemId(&templateId).
				SetQuantity(uint16(params.Quantity)).
				SetItemSnapshot(AssetData{
					Quantity:      params.Quantity,
					OwnerId:       params.CharacterId,
					Owner:         params.Owner,
					Flag:          params.Flags,
					Strength:      params.Strength,
					Dexterity:     params.Dexterity,
					Intelligence:  params.Intelligence,
					Luck:          params.Luck,
					Hp:            params.HP,
					Mp:            params.MP,
					WeaponAttack:  params.WeaponAttack,
					MagicAttack:   params.MagicAttack,
					WeaponDefense: params.WeaponDefense,
					MagicDefense:  params.MagicDefense,
					Accuracy:      params.Accuracy,
					Avoidability:  params.Avoidability,
					Hands:         params.Hands,
					Speed:         params.Speed,
					Jump:          params.Jump,
					Slots:         params.Slots,
					Level:         params.Level,
					Experience:    params.ItemExp,
				})
		}

		m, err := b.Build()
		if err != nil {
			return err
		}
		created, err := Create(tx)(m)
		if err != nil {
			return err
		}
		result = created
		return nil
	})
	if terr != nil {
		return Model{}, terr
	}
	return result, nil
}

// ReleaseCustody transitions the parcel row to received AND releases custody
// in one transaction — the status change and the custody release are the
// same fact (design §4.3). recipientId must match the row's recipient; a
// mismatch returns ErrNotRecipient and leaves the row pending. A replayed
// delivery (the row is no longer pending by the time this call runs) is not
// an error — it returns the row's current snapshot with ErrAlreadyReleased so
// the caller re-acks success without re-emitting the RELEASED status event.
func (p *ProcessorImpl) ReleaseCustody(parcelId uuid.UUID, recipientId uint32) (Model, error) {
	now := p.now()
	var result Model
	var alreadyReleased bool
	terr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		m, err := ById(parcelId)(tx)()
		if err != nil {
			return ErrNotFound
		}
		if m.RecipientId() != recipientId {
			return ErrNotRecipient
		}
		if m.Status() != StatusPending {
			alreadyReleased = true
			result = m
			return nil
		}
		rows, err := UpdateStatusIfPending(tx)(parcelId, StatusReceived, now)
		if err != nil {
			return err
		}
		if rows == 0 {
			// Lost a race to a concurrent release between the read above and
			// this write; identical outcome to a replay.
			alreadyReleased = true
		}
		result, err = ById(parcelId)(tx)()
		return err
	})
	if terr != nil {
		return Model{}, terr
	}
	if alreadyReleased {
		return result, ErrAlreadyReleased
	}
	return result, nil
}

// RestoreCustody un-resolves a parcel released by ReleaseCustody whose
// downstream accept_to_character then failed — the compensating inverse of
// ReleaseCustody (received -> pending, clearing resolvedAt). Idempotent: 0
// rows affected (the row was never released, or a prior delivery already
// restored it) is success, not an error — RestoreParcel is dispatched
// fire-and-forget by the orchestrator's compensator and no consumer awaits a
// success ack.
func (p *ProcessorImpl) RestoreCustody(parcelId uuid.UUID) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return tx.Model(&Entity{}).
			Where("id = ? AND status = ?", parcelId, StatusReceived).
			Updates(map[string]interface{}{
				"status":      StatusPending,
				"resolved_at": nil,
			}).Error
	})
}

// RemoveCustody hard-deletes a still-pending parcel row created by a late
// ACCEPT_TO_PARCEL after its saga already compensated — the compensating
// inverse of AcceptCustody. The delete is guarded to status=pending so a
// parcel already received in the interim is left untouched. Idempotent: 0
// rows affected is success, not an error — RemoveParcel is dispatched
// fire-and-forget by the orchestrator's compensator and no consumer awaits a
// success ack.
func (p *ProcessorImpl) RemoveCustody(parcelId uuid.UUID) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return tx.Unscoped().
			Where("id = ? AND status = ?", parcelId, StatusPending).
			Delete(&Entity{}).Error
	})
}
