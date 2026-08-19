package parcel

import (
	"strconv"
	"time"
)

// RestModel is the JSON:API representation of a parcel in Duey's custody —
// every Model field, tagged for the wire. This is the custody API reached
// AFTER a send request has already been validated (meso ceiling, level meso
// limit, mailbox capacity, message length all live upstream, in
// atlas-channel's parcel.ValidateSend and the lookup-requiring checks that
// wrap it — this REST surface does not repeat those checks).
type RestModel struct {
	Id string `json:"-"`

	WorldId byte `json:"worldId"`

	SenderId        uint32 `json:"senderId"`
	SenderAccountId uint32 `json:"senderAccountId"`
	SenderName      string `json:"senderName"`

	RecipientId        uint32 `json:"recipientId"`
	RecipientAccountId uint32 `json:"recipientAccountId"`

	Message    string `json:"message"`
	MesoAmount uint32 `json:"mesoAmount"`
	FeePaid    uint32 `json:"feePaid"`

	ItemId       *uint32   `json:"itemId,omitempty"`
	ItemType     byte      `json:"itemType"`
	Quantity     uint16    `json:"quantity"`
	ItemSnapshot AssetData `json:"itemSnapshot"`

	Status   string `json:"status"`
	Quick    bool   `json:"quick"`
	Returned bool   `json:"returned"`

	CreatedAt    time.Time  `json:"createdAt"`
	ReceivableAt time.Time  `json:"receivableAt"`
	ExpiresAt    time.Time  `json:"expiresAt"`
	ResolvedAt   *time.Time `json:"resolvedAt,omitempty"`
	LastNotified *time.Time `json:"lastNotified,omitempty"`
}

func (r RestModel) GetName() string {
	return "parcels"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// Transform builds the wire representation of a parcel from its domain Model.
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:                 m.Id().String(),
		WorldId:            byte(m.WorldId()),
		SenderId:           m.SenderId(),
		SenderAccountId:    m.SenderAccountId(),
		SenderName:         m.SenderName(),
		RecipientId:        m.RecipientId(),
		RecipientAccountId: m.RecipientAccountId(),
		Message:            m.Message(),
		MesoAmount:         m.MesoAmount(),
		FeePaid:            m.FeePaid(),
		ItemId:             m.ItemId(),
		ItemType:           m.ItemType(),
		Quantity:           m.Quantity(),
		ItemSnapshot:       m.ItemSnapshot(),
		Status:             m.Status(),
		Quick:              m.Quick(),
		Returned:           m.Returned(),
		CreatedAt:          m.CreatedAt(),
		ReceivableAt:       m.ReceivableAt(),
		ExpiresAt:          m.ExpiresAt(),
		ResolvedAt:         m.ResolvedAt(),
		LastNotified:       m.LastNotified(),
	}, nil
}

// parcelStatusRestModel is a narrow, one-attribute resource answering "does
// this character have a pending parcel" — a single round trip for
// task-26's world-transfer gate 12, rather than a full mailbox fetch.
type parcelStatusRestModel struct {
	Id       string `json:"-"`
	InFlight bool   `json:"inFlight"`
}

func (r parcelStatusRestModel) GetName() string {
	return "parcel-statuses"
}

func (r parcelStatusRestModel) GetID() string {
	return r.Id
}

func (r *parcelStatusRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func transformParcelStatus(characterId uint32, inFlight bool) (parcelStatusRestModel, error) {
	return parcelStatusRestModel{
		Id:       strconv.FormatUint(uint64(characterId), 10),
		InFlight: inFlight,
	}, nil
}
