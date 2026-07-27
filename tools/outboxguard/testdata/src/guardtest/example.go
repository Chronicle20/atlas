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

func good(db *gorm.DB) error { // want good:"emitsDirect"
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

// emitHelper is a concrete package-local function that constructs the direct
// producer in its own control flow (not inside a nested closure). Calling it
// inside a tx closure hides a direct emit one level deep — the class the
// lexical-only guard missed.
func emitHelper() { // want emitHelper:"emitsDirect"
	_ = producer.ProviderImpl(nil)
}

// badTransitive calls the concrete emitHelper inside the tx closure; the guard
// follows the concrete call and flags it at the call site.
func badTransitive(db *gorm.DB) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		emitHelper() // want "emitHelper reaches a direct producer.ProviderImpl and runs inside a DB transaction closure"
		return nil
	})
}

// emitHelper2 is a second concrete hop (emitHelper2 -> emitHelper); the fixed
// point marks it too, so calling it in a tx closure is still flagged.
func emitHelper2() { emitHelper() } // want emitHelper2:"emitsDirect"

func badTransitiveTwoHops(db *gorm.DB) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		emitHelper2() // want "emitHelper2 reaches a direct producer.ProviderImpl and runs inside a DB transaction closure"
		return nil
	})
}

// sagaEmitter models the dependency-injection seam: a concrete implementation
// may emit direct, but the injected value could just as well be a tx-bound
// outbox emitter, so the guard must NOT follow interface-method calls.
type sagaEmitter interface {
	Create() error
}

type directEmitter struct{}

func (directEmitter) Create() error { // want Create:"emitsDirect"
	_ = producer.ProviderImpl(nil)
	return nil
}

// interfaceSeam calls Create through the sagaEmitter interface inside a tx
// closure. The concrete directEmitter emits direct, but resolving that is
// beyond static reach (the emitter is injected), so this must NOT be flagged —
// mirrors the atlas-mts PlaceBid/Cancel escrow-saga fix (WithSagaEmitter).
func interfaceSeam(db *gorm.DB, e sagaEmitter) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return e.Create()
	})
}

// deferredHelper references producer.ProviderImpl only inside a nested closure
// (deferred/post-tx), so it must NOT be marked as emitting; calling it inside a
// tx closure is not flagged.
func deferredHelper() func() {
	return func() { _ = producer.ProviderImpl(nil) }
}

func callsDeferredHelperInTx(db *gorm.DB) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		_ = deferredHelper()
		return nil
	})
}
