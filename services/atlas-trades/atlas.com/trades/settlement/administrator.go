package settlement

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ErrSideCount reports a record that does not have exactly two sides. A
// settlement mirrors the trade: two participants, always.
var ErrSideCount = errors.New("a settlement record must have exactly two sides")

// toEntity maps the immutable Model onto the three entity rows, stamping the
// tenant at every level. Row ids come from the Model so a caller that
// correlated on Model.Id() before the write still matches afterwards.
func toEntity(t tenant.Model, m Model) Entry {
	sides := make([]Side, 0, len(m.Sides()))
	for _, s := range m.Sides() {
		items := make([]ItemRow, 0, len(s.Items()))
		for _, i := range s.Items() {
			items = append(items, ItemRow{
				Id:            uuid.New(),
				TenantId:      t.Id(),
				SideId:        s.Id(),
				EscrowId:      i.EscrowId(),
				InventoryType: i.InventoryType(),
				SourceSlot:    i.SourceSlot(),
				AssetId:       i.AssetId(),
				TemplateId:    i.TemplateId(),
				Quantity:      i.Quantity(),
			})
		}
		sides = append(sides, Side{
			Id:            s.Id(),
			TenantId:      t.Id(),
			EntryId:       m.Id(),
			Position:      s.Position(),
			CharacterId:   s.CharacterId(),
			CharacterName: s.CharacterName(),
			MesoStaged:    s.MesoStaged(),
			MesoTax:       s.MesoTax(),
			MesoDelivered: s.MesoDelivered(),
			Items:         items,
		})
	}
	return Entry{
		Id:            m.Id(),
		TenantId:      t.Id(),
		TenantRegion:  t.Region(),
		TenantMajor:   t.MajorVersion(),
		TenantMinor:   t.MinorVersion(),
		TransactionId: m.TransactionId(),
		RoomId:        m.RoomId(),
		Handle:        m.Handle(),
		RoomType:      m.RoomType(),
		WorldId:       m.Field().WorldId(),
		ChannelId:     m.Field().ChannelId(),
		MapId:         m.Field().MapId(),
		Instance:      m.Field().Instance(),
		OwnerId:       m.OwnerId(),
		VisitorId:     m.VisitorId(),
		SubmittedAt:   m.SubmittedAt(),
		Sides:         sides,
	}
}

// create writes the record, its two sides and their items in ONE transaction.
// database.ExecuteTransaction JOINS the caller's transaction when there already
// is one, which is the whole point here: the settlement processor is called
// from inside the command's transaction, so the durable record and the outbox
// row carrying the saga command commit together or not at all.
func create(db *gorm.DB, t tenant.Model) func(m Model) (Model, error) {
	return func(m Model) (Model, error) {
		if len(m.Sides()) != 2 {
			return Model{}, fmt.Errorf("%w: got %d", ErrSideCount, len(m.Sides()))
		}
		e := toEntity(t, m)
		if err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			return tx.Create(&e).Error
		}); err != nil {
			return Model{}, err
		}
		return Make(e)
	}
}

// deleteByTransactionId removes the record and its children once the terminal
// status has been handled, so unfinished settlements never accumulate. It
// reports whether THIS call deleted the record.
//
// Deleting an absent record is NOT an error, it is a `false`: reconciliation
// and the live status consumer can both reach the same settlement, and
// whichever arrives second must be told it lost rather than be failed. That
// boolean is the terminal path's arbiter — two concurrent deliveries serialise
// on this delete, so exactly one of them sees a row to remove.
func deleteByTransactionId(db *gorm.DB, tenantId uuid.UUID) func(transactionId uuid.UUID) (bool, error) {
	return func(transactionId uuid.UUID) (bool, error) {
		deleted := false
		err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			var e Entry
			err := tx.Where("tenant_id = ? AND transaction_id = ?", tenantId, transactionId).First(&e).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			var sides []Side
			if err = tx.Where("tenant_id = ? AND entry_id = ?", tenantId, e.Id).Find(&sides).Error; err != nil {
				return err
			}
			for _, s := range sides {
				if err = tx.Where("tenant_id = ? AND side_id = ?", tenantId, s.Id).Delete(&ItemRow{}).Error; err != nil {
					return err
				}
			}
			if err = tx.Where("tenant_id = ? AND entry_id = ?", tenantId, e.Id).Delete(&Side{}).Error; err != nil {
				return err
			}
			result := tx.Where("tenant_id = ? AND id = ?", tenantId, e.Id).Delete(&Entry{})
			if result.Error != nil {
				return result.Error
			}
			deleted = result.RowsAffected > 0
			return nil
		})
		return deleted, err
	}
}
