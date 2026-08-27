package asset

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id            uint32    `gorm:"primaryKey;autoIncrement:true"`
	TenantId      uuid.UUID `gorm:"not null"`
	CompartmentId uuid.UUID `gorm:"not null"`
	CashId        int64     `gorm:"not null"`
	TemplateId    uint32    `gorm:"not null"`
	CommodityId   uint32    `gorm:"not null;default:0"`
	// Currency is the wallet bucket (wallet.Model.Balance's convention: 1 =
	// credit/NX, 2 = Maple Points, anything else = prepaid) this asset was
	// purchased with -- recorded so a locker rebate (task-240 task 11) knows
	// which bucket to credit back, instead of guessing.
	//
	// Purchase (cashshop/processor.go) never persists the raw wire currency
	// here -- it persists effectivePurchaseCurrency(currency), which maps
	// 1/2 unchanged and every other wire value (including 0, BUY_NORMAL's
	// wire currency) to walletCurrencyPrepaid (3). That normalization is
	// what makes a stored 0 UNAMBIGUOUS: it means EITHER (a) this row
	// predates the Currency column, or (b) this asset was created on a
	// gift/reward/surprise path that was never bought with currency at all
	// (astP.Create/CreateWithCashId called with a literal 0) -- and NEVER
	// "a genuine purchase whose bucket was prepaid" (fix round 1; a prior,
	// wrong version of this comment claimed 0 already meant the default
	// bucket for an ordinary buy -- it did not, because wallet.Model routes
	// 0 to prepaid, not credit, and Purchase used to persist the raw wire
	// value unchanged). A rebate therefore safely treats a stored 0 as the
	// ordinary credit/NX bucket (the user's ruling, C2) and every other
	// stored value as itself, with no further guessing.
	Currency    uint32    `gorm:"not null;default:0"`
	Quantity    uint32    `gorm:"not null"`
	Flag        uint16    `gorm:"not null"`
	PetId       uint32    `gorm:"not null;default:0"`
	PurchasedBy uint32    `gorm:"not null"`
	Expiration  time.Time `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null"`
	// GiftFrom is the sender's character name for a GIFT purchase (task-240
	// task 13), empty for every other asset. Bounded at 13 characters -- the
	// padded encode width of CashInventoryItem.GiftFrom
	// (libs/atlas-packet/cash/clientbound/shop_inventory.go:35).
	GiftFrom string `gorm:"size:13;not null;default:''"`
	// GiftMessage is the sender's message for a GIFT purchase, empty for
	// every other asset. Bounded at 73 characters -- the padded encode width
	// of GiftListEntry.Text
	// (libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go:43).
	GiftMessage string `gorm:"size:73;not null;default:''"`
	// GiftAcknowledged records whether the gift list carrying this asset has
	// already been PRESENTED to the recipient (a LOAD_GIFT_SUCCESS announce
	// has fired for it) -- task-240 Defect H. This is NOT "the recipient
	// clicked OK on the modal": the client sends nothing when the modal is
	// cancelled, so the only trigger that fires exactly once per
	// presentation is the announce itself, and this flag is drained at that
	// point, not on acknowledgement. AutoMigrate lands the column default
	// (false) on every existing row, so a locker asset created before this
	// field existed is treated as not-yet-presented, which is correct: it
	// has never been announced under this flag's regime.
	GiftAcknowledged bool `gorm:"not null;default:false"`
	// GiftNoteSent records whether the gift-forward NOTE_ACTION SEND for
	// this asset has already minted its free note to the gifter (task-240
	// Defect I). This is a SECOND, independent flag from GiftAcknowledged --
	// it answers "has its note been sent?", not "was this presented?", and
	// it drains at a different moment: the note *send*, not the
	// LOAD_GIFT_SUCCESS announce. By the time a legitimate note arrives,
	// GiftAcknowledged has already been drained by the announce, so
	// handleNoteGiftForward must gate on THIS flag, never on
	// GiftAcknowledged. Without it, a modified client that re-emits
	// giftFlag=1 for a gift it still holds could mint unlimited free notes.
	// AutoMigrate lands the column default (false) on every existing row.
	//
	// Known limitation, not fixed here: the mark-sent command is
	// asynchronous (a Kafka round trip), so two acknowledgement packets
	// racing inside that window can both pass this gate before either
	// write lands. This narrows the exposure from unbounded to a single
	// race; closing it fully would need a synchronous write on the packet
	// path, which no other cash-shop flow in this service does.
	GiftNoteSent bool           `gorm:"not null;default:false"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (e Entity) TableName() string {
	return "cash_assets"
}

func Make(e Entity) (Model, error) {
	return NewBuilder(e.CompartmentId, e.TemplateId).
		SetId(e.Id).
		SetCashId(e.CashId).
		SetCommodityId(e.CommodityId).
		SetCurrency(e.Currency).
		SetQuantity(e.Quantity).
		SetFlag(e.Flag).
		SetPetId(e.PetId).
		SetPurchasedBy(e.PurchasedBy).
		SetExpiration(e.Expiration).
		SetCreatedAt(e.CreatedAt).
		SetGiftFrom(e.GiftFrom).
		SetGiftMessage(e.GiftMessage).
		SetGiftAcknowledged(e.GiftAcknowledged).
		SetGiftNoteSent(e.GiftNoteSent).
		Build(), nil
}
