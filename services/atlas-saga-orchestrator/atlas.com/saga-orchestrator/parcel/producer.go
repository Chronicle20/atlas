package parcel

import (
	parcelmsg "atlas-saga-orchestrator/kafka/message/parcel"
	parcelCustody "atlas-saga-orchestrator/kafka/message/parcel/custody"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// AcceptToParcelProvider creates an ACCEPT_TO_PARCEL command for the
// atlas-parcel custody consumer. Keyed by the parcel id so all custody
// commands for a parcel are ordered.
func AcceptToParcelProvider(transactionId uuid.UUID, params AcceptToParcelParams) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(params.ParcelId.ID()))
	value := &parcelCustody.Command[parcelCustody.AcceptToParcelCommandBody]{
		TransactionId: transactionId,
		Type:          parcelCustody.CommandAcceptToParcel,
		Body: parcelCustody.AcceptToParcelCommandBody{
			ParcelId:           params.ParcelId,
			CharacterId:        params.CharacterId,
			WorldId:            params.WorldId,
			SenderAccountId:    params.SenderAccountId,
			SenderName:         params.SenderName,
			RecipientId:        params.RecipientId,
			RecipientAccountId: params.RecipientAccountId,
			RecipientName:      params.RecipientName,
			MesoAmount:         params.MesoAmount,
			FeePaid:            params.FeePaid,
			Quick:              params.Quick,
			Message:            params.Message,
			ReceivableAt:       params.ReceivableAt,
			ExpiresAt:          params.ExpiresAt,
			HasItem:            params.HasItem,
			TemplateId:         params.TemplateId,
			Quantity:           params.Quantity,
			Strength:           params.Strength,
			Dexterity:          params.Dexterity,
			Intelligence:       params.Intelligence,
			Luck:               params.Luck,
			HP:                 params.HP,
			MP:                 params.MP,
			WeaponAttack:       params.WeaponAttack,
			MagicAttack:        params.MagicAttack,
			WeaponDefense:      params.WeaponDefense,
			MagicDefense:       params.MagicDefense,
			Accuracy:           params.Accuracy,
			Avoidability:       params.Avoidability,
			Hands:              params.Hands,
			Speed:              params.Speed,
			Jump:               params.Jump,
			Slots:              params.Slots,
			Level:              params.Level,
			ItemLevel:          params.ItemLevel,
			ItemExp:            params.ItemExp,
			RingId:             params.RingId,
			ViciousCount:       params.ViciousCount,
			Flags:              params.Flags,
			Owner:              params.Owner,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ReleaseFromParcelProvider creates a RELEASE_FROM_PARCEL command. Keyed by
// the parcel id so replays of the same release are ordered.
func ReleaseFromParcelProvider(transactionId uuid.UUID, parcelId uuid.UUID, recipientId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(parcelId.ID()))
	value := &parcelCustody.Command[parcelCustody.ReleaseFromParcelCommandBody]{
		TransactionId: transactionId,
		Type:          parcelCustody.CommandReleaseFromParcel,
		Body: parcelCustody.ReleaseFromParcelCommandBody{
			ParcelId:    parcelId,
			RecipientId: recipientId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// RestoreParcelProvider creates a RESTORE_PARCEL command (the compensating
// inverse of RELEASE_FROM_PARCEL). Keyed by the parcel id so replays of the
// same restore are ordered.
func RestoreParcelProvider(transactionId uuid.UUID, parcelId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(parcelId.ID()))
	value := &parcelCustody.Command[parcelCustody.RestoreParcelCommandBody]{
		TransactionId: transactionId,
		Type:          parcelCustody.CommandRestoreParcel,
		Body: parcelCustody.RestoreParcelCommandBody{
			ParcelId: parcelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// RemoveParcelProvider creates a REMOVE_PARCEL command (the compensating
// inverse of ACCEPT_TO_PARCEL). Keyed by the parcel id so replays of the same
// removal are ordered.
func RemoveParcelProvider(transactionId uuid.UUID, parcelId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(parcelId.ID()))
	value := &parcelCustody.Command[parcelCustody.RemoveParcelCommandBody]{
		TransactionId: transactionId,
		Type:          parcelCustody.CommandRemoveParcel,
		Body: parcelCustody.RemoveParcelCommandBody{
			ParcelId: parcelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ShowParcelCommandProvider creates a SHOW_PARCEL command for atlas-channel.
// Keyed by the character id, mirroring ShowStorageCommandProvider.
func ShowParcelCommandProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, npcId uint32, quick bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &parcelmsg.ShowParcelCommand{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		ChannelId:     ch.Id(),
		CharacterId:   characterId,
		NpcId:         npcId,
		Quick:         quick,
		Type:          parcelmsg.CommandTypeShowParcel,
	}
	return producer.SingleMessageProvider(key, value)
}
