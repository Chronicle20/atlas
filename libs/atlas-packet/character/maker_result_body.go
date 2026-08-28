package character

import (
	"context"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// Fixed operation keys for the CUserLocal::OnMakerResult (MAKER_RESULT)
// dispatcher arms. Each body function below resolves its OWN const from the
// tenant "operations" table — the key is never a parameter, so a caller can
// never pick the arm's mode (DOM-25 / INV-3).
const (
	MakerResultOperationCreate            = "CREATE"
	MakerResultOperationCreateWithUpgrade = "CREATE_WITH_UPGRADE"
	MakerResultOperationMonsterCrystal    = "MONSTER_CRYSTAL"
	MakerResultOperationDisassemble       = "DISASSEMBLE"
	MakerResultOperationFailed            = "FAILED"
)

// MakerResultCreateBody builds the mode-1 CREATE arm.
func MakerResultCreateBody(result uint32, noItemAwarded bool, targetItemId uint32, itemNum uint32, materials []clientbound.MakerMaterial, gemItemIds []uint32, catalystUsed bool, catalystItemId uint32, mesoCost uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MakerResultOperationCreate, func(m byte) packet.Encoder {
		return clientbound.NewMakerResultCreate(m, result, noItemAwarded, targetItemId, itemNum, materials, gemItemIds, catalystUsed, catalystItemId, mesoCost)
	})
}

// MakerResultCreateWithUpgradeBody builds the mode-2 CREATE_WITH_UPGRADE arm.
func MakerResultCreateWithUpgradeBody(result uint32, noItemAwarded bool, targetItemId uint32, itemNum uint32, materials []clientbound.MakerMaterial, gemItemIds []uint32, catalystUsed bool, catalystItemId uint32, mesoCost uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MakerResultOperationCreateWithUpgrade, func(m byte) packet.Encoder {
		return clientbound.NewMakerResultCreateWithUpgrade(m, result, noItemAwarded, targetItemId, itemNum, materials, gemItemIds, catalystUsed, catalystItemId, mesoCost)
	})
}

// MakerResultMonsterCrystalBody builds the mode-3 MONSTER_CRYSTAL arm.
func MakerResultMonsterCrystalBody(result uint32, crystalItemId uint32, leftoverItemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MakerResultOperationMonsterCrystal, func(m byte) packet.Encoder {
		return clientbound.NewMakerResultMonsterCrystal(m, result, crystalItemId, leftoverItemId)
	})
}

// MakerResultDisassembleBody builds the mode-4 DISASSEMBLE arm.
func MakerResultDisassembleBody(result uint32, disassembledItemId uint32, crystals []clientbound.MakerMaterial, mesoCost uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MakerResultOperationDisassemble, func(m byte) packet.Encoder {
		return clientbound.NewMakerResultDisassemble(m, result, disassembledItemId, crystals, mesoCost)
	})
}

// MakerResultFailedBody builds the bodyless FAILED arm.
//
// This is the one arm that does NOT go through WithResolvedCode, and the reason
// is the wire, not convenience: the client's nResult guard (see
// clientbound.MakerResultFailed) stops reading before the nMode Decode4, so this
// arm has no mode field at all. There is consequently no per-version mode value
// for FAILED in the family's dispatcher YAML, and resolving the key anyway would
// make ResolveCode log a misconfiguration and hand back its 99 sentinel on every
// send — for a byte that is then discarded. Constructing the arm directly is the
// honest encoding; MakerResultOperationFailed is retained as the arm's stable
// identifier for the family's bookkeeping.
//
// It is emphatically NOT a caller-selected mode: like the other four, the arm is
// fixed at the call site and no parameter can change it (INV-3).
func MakerResultFailedBody(result uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return clientbound.NewMakerResultFailed(result).Encode(l, ctx)(options)
		}
	}
}
