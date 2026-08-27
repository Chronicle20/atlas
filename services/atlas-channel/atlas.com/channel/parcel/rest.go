package parcel

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel is atlas-channel's projection of atlas-parcel's custody
// resource (services/atlas-parcel/atlas.com/parcel/parcel/rest.go
// RestModel). It carries what the send-side pre-flight and Task 18's
// receive/discard arms need — not the full item snapshot, which is the
// saga expansion's concern (the piece that reconstitutes the item into an
// inventory), never atlas-channel's.
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

	ItemId   *uint32 `json:"itemId,omitempty"`
	ItemType byte    `json:"itemType"`
	Quantity uint16  `json:"quantity"`

	Status   string `json:"status"`
	Quick    bool   `json:"quick"`
	Returned bool   `json:"returned"`

	// CreatedAt is the parcel's creation timestamp — Task 21's only reader.
	// It is NOT the OPEN packet's PARCEL.expiresAt (+21 FILETIME); that wire
	// field carries ExpiresAt below (task-23 / RISK-4 resolution,
	// docs/tasks/task-241-duey-parcel-delivery/context.md §11 — the client's
	// receive guard divides unsigned, so +21 must be a future deadline, not
	// the send time).
	CreatedAt time.Time `json:"createdAt"`

	// ReceivableAt gates Task 18's receive pre-flight (design §7.2): a
	// pending parcel addressed to the caller is not receivable until this
	// passes. Not carried by Task 17's send-side pre-flight, which never
	// reads an individual parcel's timing.
	ReceivableAt time.Time `json:"receivableAt"`

	// ExpiresAt is the OPEN packet's PARCEL.expiresAt (+21 FILETIME,
	// task-23 / RISK-4 resolution). atlas-parcel's REST already emits this
	// (services/atlas-parcel/.../parcel/rest.go), so no producer-side change
	// was needed.
	ExpiresAt time.Time `json:"expiresAt"`

	// LastNotified is nil until either Task 21's SHOW_PARCEL consumer or
	// Task 24's atlas-parcel notification sweep stamps it — both writers
	// share the single nullable column and its one meaning ("the player has
	// been told about this parcel once"). Task 21 reads it to split the
	// OPEN packet's mailbox from its "new arrivals" second list (FR-24).
	LastNotified *time.Time `json:"lastNotified,omitempty"`
}

func (r RestModel) GetName() string { return "parcels" }

func (r RestModel) GetID() string { return r.Id }

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// Required JSON:API relationship stubs (libs/atlas-rest gotcha): api2go
// errors out decoding any response unless the target implements these, even
// with no relationships present.
func (r *RestModel) SetToOneReferenceID(_, _ string) error { return nil }

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// Transform converts the domain Model into the wire RestModel.
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:                 m.id.String(),
		WorldId:            byte(m.worldId),
		SenderId:           m.senderId,
		SenderAccountId:    m.senderAccountId,
		SenderName:         m.senderName,
		RecipientId:        m.recipientId,
		RecipientAccountId: m.recipientAccountId,
		Message:            m.message,
		MesoAmount:         m.mesoAmount,
		FeePaid:            m.feePaid,
		ItemId:             m.itemId,
		ItemType:           m.itemType,
		Quantity:           m.quantity,
		Status:             m.status,
		Quick:              m.quick,
		Returned:           m.returned,
		CreatedAt:          m.createdAt,
		ReceivableAt:       m.receivableAt,
		ExpiresAt:          m.expiresAt,
		LastNotified:       m.lastNotified,
	}, nil
}

// Extract builds a Model from the wire RestModel.
func Extract(rm RestModel) (Model, error) {
	id, err := uuid.Parse(rm.Id)
	if err != nil {
		return Model{}, err
	}
	return Model{
		id:                 id,
		worldId:            world.Id(rm.WorldId),
		senderId:           rm.SenderId,
		senderAccountId:    rm.SenderAccountId,
		senderName:         rm.SenderName,
		recipientId:        rm.RecipientId,
		recipientAccountId: rm.RecipientAccountId,
		message:            rm.Message,
		mesoAmount:         rm.MesoAmount,
		feePaid:            rm.FeePaid,
		itemId:             rm.ItemId,
		itemType:           rm.ItemType,
		quantity:           rm.Quantity,
		status:             rm.Status,
		quick:              rm.Quick,
		returned:           rm.Returned,
		createdAt:          rm.CreatedAt,
		receivableAt:       rm.ReceivableAt,
		expiresAt:          rm.ExpiresAt,
		lastNotified:       rm.LastNotified,
	}, nil
}
