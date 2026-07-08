package rps

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/rps/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// RPSGameMode is the operations-table key type for the RPSGame writer's mode
// byte. Only the three modes atlas-rps emits are defined here (OPEN, RESULT,
// END) — see docs/packets/dispatchers/rps_game.yaml and
// docs/tasks/task-132-rps-npc-game/ida-rps-clientbound.md §0/§6 for the full
// client-side mode set (6/7/9/10/12/14 exist client-side but are never sent).
type RPSGameMode = string

const (
	RPSGameModeOpen   RPSGameMode = "OPEN"
	RPSGameModeResult RPSGameMode = "RESULT"
	RPSGameModeEnd    RPSGameMode = "END"
)

// RPSGameOpenBody constructs the OPEN arm body function. ante is the
// participation fee; the mode byte is resolved per-tenant from the
// operations table (rps_game.yaml OPEN row), never hard-coded.
func RPSGameOpenBody(ante uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", RPSGameModeOpen, func(mode byte) packet.Encoder {
		return clientbound.NewRPSGameOpen(mode, ante)
	})
}

// RPSGameResultBody constructs the RESULT arm body function. npcThrow is the
// NPC's R/P/S throw; straightVictoryCount is the SIGNED consecutive-win
// count (negative signals game-over/final — see the IDA note). The mode byte
// is resolved per-tenant from the operations table (rps_game.yaml RESULT row).
func RPSGameResultBody(npcThrow byte, straightVictoryCount int8) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", RPSGameModeResult, func(mode byte) packet.Encoder {
		return clientbound.NewRPSGameResult(mode, npcThrow, straightVictoryCount)
	})
}

// RPSGameEndBody constructs the END (CLOSE) arm body function. No arm data —
// mode byte only. The mode byte is resolved per-tenant from the operations
// table (rps_game.yaml END row).
func RPSGameEndBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", RPSGameModeEnd, func(mode byte) packet.Encoder {
		return clientbound.NewRPSGameEnd(mode)
	})
}
