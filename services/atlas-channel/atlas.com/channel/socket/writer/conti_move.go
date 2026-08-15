package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// ContiMoveKey selects which state/subState pair to resolve from the
// tenant's ContiMove writer options table (DOM-25). state/subState are
// client wire bytes -- this repo already litigated (task-102/task-103) that
// such values are resolved per tenant through the writer options registry,
// never carried as free-form config from whatever domain event triggers the
// write, even when the value happens to be version-stable per the IDB.
type ContiMoveKey string

const (
	// ContiMoveShow resolves the state/subState pair for showing a visual
	// (the SHOW_STATE/SHOW_SUB_STATE keys in the tenant's ContiMove options).
	ContiMoveShow ContiMoveKey = "SHOW"
	// ContiMoveHide resolves the state/subState pair for retiring a visual
	// (the HIDE_STATE/HIDE_SUB_STATE keys in the tenant's ContiMove options).
	ContiMoveHide ContiMoveKey = "HIDE"
)

// ContiMoveBody resolves key's state/subState pair from the tenant's
// ContiMove writer options table and encodes the wire packet. A resolve
// miss falls back to ResolveCode's loud 99 sentinel (logged, will likely
// crash the client) rather than guessing a value -- the same failure mode
// every other options-resolved writer in this package uses.
func ContiMoveBody(key ContiMoveKey) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			state := atlas_packet.ResolveCode(l, options, "operations", string(key)+"_STATE")
			subState := atlas_packet.ResolveCode(l, options, "operations", string(key)+"_SUB_STATE")
			return fieldcb.NewContiMove(state, subState).Encode(l, ctx)(options)
		}
	}
}
