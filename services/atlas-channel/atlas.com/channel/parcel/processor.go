package parcel

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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

	// ReceivableAt gates Task 18's receive pre-flight (design §7.2): a
	// pending parcel addressed to the caller is not receivable until this
	// passes. Not carried by Task 17's send-side pre-flight, which never
	// reads an individual parcel's timing.
	ReceivableAt time.Time `json:"receivableAt"`
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

// Model is atlas-channel's read-only view of one parcel in Duey's custody.
type Model struct {
	id                 uuid.UUID
	worldId            world.Id
	senderId           uint32
	senderAccountId    uint32
	senderName         string
	recipientId        uint32
	recipientAccountId uint32
	message            string
	mesoAmount         uint32
	feePaid            uint32
	itemId             *uint32
	itemType           byte
	quantity           uint16
	status             string
	quick              bool
	returned           bool
	receivableAt       time.Time
}

func (m Model) Id() uuid.UUID              { return m.id }
func (m Model) WorldId() world.Id          { return m.worldId }
func (m Model) SenderId() uint32           { return m.senderId }
func (m Model) SenderAccountId() uint32    { return m.senderAccountId }
func (m Model) SenderName() string         { return m.senderName }
func (m Model) RecipientId() uint32        { return m.recipientId }
func (m Model) RecipientAccountId() uint32 { return m.recipientAccountId }
func (m Model) Message() string            { return m.message }
func (m Model) MesoAmount() uint32         { return m.mesoAmount }
func (m Model) FeePaid() uint32            { return m.feePaid }
func (m Model) ItemId() *uint32            { return m.itemId }
func (m Model) ItemType() byte             { return m.itemType }
func (m Model) Quantity() uint16           { return m.quantity }
func (m Model) Status() string             { return m.status }
func (m Model) Quick() bool                { return m.quick }
func (m Model) Returned() bool             { return m.returned }
func (m Model) ReceivableAt() time.Time    { return m.receivableAt }

// WireId projects a parcel's atlas-parcel uuid.UUID identity onto the
// wire's uint32 parcelId — the client's PARCEL struct's `+0 uint32 parcelId`,
// echoed verbatim by CTabReceive::ReceiveParcel/DiscardParcel (design §5.3).
// It resolves the client's wire-format uint32 parcelId against atlas-parcel's
// uuid.UUID row identity by taking the first 4 bytes of the id, big-endian.
// This is a deliberate, self-contained engineering choice for a wire detail
// the design doc explicitly leaves to implementation ("the exact byte
// layout... is derived during implementation", §5.3). Every caller that
// projects a parcel id onto the wire — DUEY_ACTION RECEIVE/DISCARD's
// resolution and Model.ToPacket()'s emission of the OPEN list body alike —
// MUST use this function rather than re-deriving the projection, or the
// two directions will silently disagree.
func WireId(id uuid.UUID) uint32 {
	return binary.BigEndian.Uint32(id[:4])
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
		receivableAt:       rm.ReceivableAt,
	}, nil
}

// Processor is atlas-channel's read-only view of atlas-parcel's custody
// resource.
type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

// GetForRecipient lists recipientId's pending mailbox in worldId.
func (p *Processor) GetForRecipient(recipientId uint32, worldId world.Id) ([]Model, error) {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestForRecipient(p.ctx, recipientId, worldId), Extract, model.Filters[Model]())()
}

// CountPending answers the mailbox-capacity check (design §6.2) — the
// length of recipientId's pending mailbox in worldId, the only pre-flight
// besides recipient resolution that leaves the channel.
//
// worldId is required here even though the brief's interface sketch shows
// CountPending(recipientId) alone: atlas-parcel's GET /parcels rejects a
// recipient filter with no world filter (services/atlas-parcel/.../
// resource.go handleGetParcels), and defaulting it internally would be
// exactly the world-0-is-not-a-sentinel mistake this branch has already
// flagged four times elsewhere. The caller already has the recipient's
// resolved world (it filtered ByNameProvider's result to it), so passing it
// through costs nothing.
func (p *Processor) CountPending(recipientId uint32, worldId world.Id) (int, error) {
	ms, err := p.GetForRecipient(recipientId, worldId)
	if err != nil {
		return 0, err
	}
	return len(ms), nil
}

// GetById retrieves a single parcel by id (Task 18's receive/discard arms).
func (p *Processor) GetById(id uuid.UUID) (Model, error) {
	rm, err := requestById(p.ctx, id)(p.l, p.ctx)
	if err != nil {
		return Model{}, err
	}
	return Extract(rm)
}

// Discard marks parcelId discarded on behalf of recipientId (design §4.4 /
// §7.3): discard is deliberately not a saga, so this is a direct
// PATCH /parcels/{id} against atlas-parcel, which owns the recipient/status
// validation server-side (parcel.ProcessorImpl.Discard).
func (p *Processor) Discard(id uuid.UUID, recipientId uint32) (Model, error) {
	rm, err := discardRequest(p.ctx, id, recipientId)(p.l, p.ctx)
	if err != nil {
		return Model{}, err
	}
	return Extract(rm)
}
