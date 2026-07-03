package guardtest

import (
	"gorm"
	"message"
	"producer"
	"txdb"
)

func bad(db *gorm.DB) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		_ = producer.ProviderImpl(nil) // want "producer.ProviderImpl inside a DB transaction closure"
		return nil
	})
}

func alsoBad(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		_ = producer.ProviderImpl(nil) // want "producer.ProviderImpl inside a DB transaction closure"
		return nil
	})
}

func good(db *gorm.DB) error {
	err := database.ExecuteTransaction(db, func(tx *gorm.DB) error { return nil })
	_ = producer.ProviderImpl(nil)
	return err
}

// deferredClosure mirrors the rejectEmit pattern (atlas-character,
// atlas-fame, atlas-cashshop): producer.ProviderImpl is only referenced
// inside a nested func literal that is captured for invocation strictly
// after the transaction returns, on the direct (non-tx) path. It must NOT
// be flagged.
func deferredClosure(db *gorm.DB) error {
	var rejectEmit func() error
	txErr := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		if tx == nil {
			rejectEmit = func() error {
				_ = producer.ProviderImpl(nil)
				return nil
			}
			return nil
		}
		return nil
	})
	if rejectEmit != nil {
		return rejectEmit()
	}
	return txErr
}

// emitArgInTx mirrors the genuine atlas-monster-book bug shape: within the
// tx closure's own body, producer.ProviderImpl is called directly as an
// argument expression to message.Emit — not inside the nested func(mb)
// literal — so it still executes as part of the transaction's control flow
// and must be flagged.
func emitArgInTx(db *gorm.DB) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return message.Emit(producer.ProviderImpl(nil))(func(mb *message.Buffer) error { // want "producer.ProviderImpl inside a DB transaction closure"
			return nil
		})
	})
}
