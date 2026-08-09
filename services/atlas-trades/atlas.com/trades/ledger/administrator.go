package ledger

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

// ErrDuplicateTransaction reports that a settlement had already been recorded
// for this (tenant, transaction) pair. create resolves it internally by
// returning the existing entry, so it only reaches a caller when the existing
// row could not be read back.
var ErrDuplicateTransaction = errors.New("a ledger entry already exists for this transaction")

// ErrSideCount reports an entry that does not have exactly two sides. The
// ledger mirrors the trade: one giver and one receiver, always (PRD §6).
var ErrSideCount = errors.New("a ledger entry must have exactly two sides")

// uniqueViolationSQLState is the PostgreSQL SQLSTATE for unique_violation. It
// is matched the same way libs/atlas-database's transient classifier matches
// its own states — by code off a *pgconn.PgError, never by message text.
const uniqueViolationSQLState = "23505"

// isDuplicateTransaction reports whether err is the (tenant_id,
// transaction_id) unique-index violation. PostgreSQL is the production driver;
// gorm.ErrDuplicatedKey covers drivers configured with TranslateError, and the
// sqlite message match covers the in-memory test database, whose driver
// reports constraint failures only as text.
func isDuplicateTransaction(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == uniqueViolationSQLState
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// toEntity maps the immutable Model onto the three entity rows, stamping
// tenantId at every level. Row ids come from the Model so a caller that
// correlated on Model.Id() before the write still matches afterwards.
func toEntity(tenantId uuid.UUID, m Model) Entry {
	sides := make([]Side, 0, len(m.Sides()))
	for _, s := range m.Sides() {
		items := make([]ItemRow, 0, len(s.Items()))
		for _, i := range s.Items() {
			row := ItemRow{
				Id:       uuid.New(),
				TenantId: tenantId,
				SideId:   s.Id(),
				ItemId:   i.ItemId(),
				Quantity: i.Quantity(),
			}
			if assetId, ok := i.AssetId(); ok {
				row.AssetId = &assetId
			}
			if referenceId, ok := i.ReferenceId(); ok {
				row.ReferenceId = &referenceId
			}
			items = append(items, row)
		}
		sides = append(sides, Side{
			Id:            s.Id(),
			TenantId:      tenantId,
			EntryId:       m.Id(),
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
		TenantId:      tenantId,
		TransactionId: m.TransactionId(),
		WorldId:       m.Field().WorldId(),
		ChannelId:     m.Field().ChannelId(),
		MapId:         m.Field().MapId(),
		RoomType:      m.RoomType(),
		SettledAt:     m.SettledAt(),
		Sides:         sides,
	}
}

// create writes the entry, its two sides and their items in ONE transaction —
// a half-written ledger row is worse than none, and database.ExecuteTransaction
// joins the caller's transaction when there already is one, so a settlement
// handler can record the trade in the same transaction as its outbox rows.
//
// It is idempotent per (tenant_id, transaction_id) (FR-5.7, design §9): a
// retried settlement saga finds the existing entry on the in-transaction read
// and returns it unchanged rather than recording the trade twice. Two settles
// racing past that read collide on the idx_trade_entry_tenant_tx unique index;
// that aborts the transaction, so the existing row is re-read on a fresh one.
func create(db *gorm.DB, tenantId uuid.UUID) func(m Model) (Model, error) {
	return func(m Model) (Model, error) {
		if len(m.Sides()) != 2 {
			return Model{}, fmt.Errorf("%w: got %d", ErrSideCount, len(m.Sides()))
		}

		var out Model
		err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			existing, rerr := byTransactionId(tx, tenantId)(m.TransactionId())
			if rerr == nil {
				out = existing
				return nil
			}
			if !errors.Is(rerr, gorm.ErrRecordNotFound) {
				return rerr
			}

			e := toEntity(tenantId, m)
			if cerr := tx.Create(&e).Error; cerr != nil {
				if isDuplicateTransaction(cerr) {
					return ErrDuplicateTransaction
				}
				return cerr
			}

			stored, merr := Make(e)
			if merr != nil {
				return merr
			}
			out = stored
			return nil
		})
		if err == nil {
			return out, nil
		}
		if !errors.Is(err, ErrDuplicateTransaction) {
			return Model{}, err
		}

		// The transaction that lost the race is aborted, so the read has to
		// run on the original handle. When that handle was itself the
		// caller's (now aborted) transaction the read fails too, and the
		// caller gets ErrDuplicateTransaction to retry on a fresh one.
		existing, rerr := byTransactionId(db, tenantId)(m.TransactionId())
		if rerr != nil {
			return Model{}, errors.Join(ErrDuplicateTransaction, rerr)
		}
		return existing, nil
	}
}
