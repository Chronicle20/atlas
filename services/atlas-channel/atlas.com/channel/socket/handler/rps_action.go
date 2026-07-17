package handler

import (
	"atlas-channel/rps"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	rpssb "github.com/Chronicle20/atlas/libs/atlas-packet/rps/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// RPSActionHandle is the config key for the CRPSGameDlg RPS_ACTION
// serverbound dispatcher (see libs/atlas-packet/rps/serverbound.RPSActionHandle).
const RPSActionHandle = rpssb.RPSActionHandle

// RPSActionMode names the tenant `operations` table keys this handler
// resolves the leading sub-op byte against. These MUST match the keys the
// Task 20 seed template defines for RPSActionHandle's operations table.
type RPSActionMode string

const (
	RPSActionModeStart    RPSActionMode = "START"
	RPSActionModeSelect   RPSActionMode = "SELECT"
	RPSActionModeUpdate   RPSActionMode = "UPDATE"
	RPSActionModeContinue RPSActionMode = "CONTINUE"
	RPSActionModeExit     RPSActionMode = "EXIT"
	RPSActionModeRetry    RPSActionMode = "RETRY"
)

// emitRPSBeginFunc/emitRPSSelectFunc/emitRPSContinueFunc/emitRPSCollectFunc are
// seams over rps.NewProcessor, swappable in tests (mirrors the door handler's
// doorsByOwnerFunc/partyMemberIdsFunc seam pattern in mystic_door_enter.go).
var emitRPSBeginFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32, worldId world.Id, channelId channel.Id) error {
	return rps.NewProcessor(l, ctx).Begin(characterId, worldId, channelId)
}

var emitRPSSelectFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32, worldId world.Id, channelId channel.Id, throw byte) error {
	return rps.NewProcessor(l, ctx).Select(characterId, worldId, channelId, throw)
}

var emitRPSContinueFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32, worldId world.Id, channelId channel.Id) error {
	return rps.NewProcessor(l, ctx).Continue(characterId, worldId, channelId)
}

var emitRPSRetryFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32, worldId world.Id, channelId channel.Id) error {
	return rps.NewProcessor(l, ctx).Retry(characterId, worldId, channelId)
}

var emitRPSCollectFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32, worldId world.Id, channelId channel.Id) error {
	return rps.NewProcessor(l, ctx).Collect(characterId, worldId, channelId)
}

// RPSActionHandleFunc decodes the CRPSGameDlg RPS_ACTION sub-op and emits the
// matching atlas-rps command. Sub-op -> command mapping (IDA-verified,
// Task 16 + atlas-rps amendment Task 17b):
//
//	START(0)           -> CommandTypeBegin (opens the first round: atlas-rps
//	                      transitions the open session to awaiting-select and
//	                      emits RoundStarted, which the channel turns into the
//	                      clientbound START_SELECT frame that enables the R/P/S
//	                      buttons - the client blocks on this before it will let
//	                      the player pick, so it is NOT a no-op)
//	SELECT(1, +throw) -> CommandTypeSelect{Throw}  (throw passed RAW, unremapped)
//	CONTINUE(3)        -> CommandTypeContinue
//	EXIT(4)            -> CommandTypeCollect (the client's only "leave" action;
//	                      there is no dedicated collect sub-op - atlas-rps's
//	                      Collect handles collect-or-forfeit by session status)
//	RETRY(5)           -> CommandTypeRetry (restart after a loss: atlas-rps
//	                      re-charges the entry fee and reopens a fresh round,
//	                      re-arming the client via START_SELECT)
//	UPDATE(2, timeout) -> no-op (TTL sweeper reaps abandoned sessions)
func RPSActionHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := rpssb.Operation{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		mode := p.Mode()

		if isRPSAction(l)(readerOptions, mode, RPSActionModeSelect) {
			sp := &rpssb.OperationSelect{}
			sp.Decode(l, ctx)(r, readerOptions)
			throw := sp.Throw()
			l.Debugf("Character [%d] selected RPS throw [%d].", s.CharacterId(), throw)
			if err := emitRPSSelectFunc(l, ctx, s.CharacterId(), s.WorldId(), s.ChannelId(), throw); err != nil {
				l.WithError(err).Errorf("Unable to emit RPS SELECT command for character [%d].", s.CharacterId())
			}
			return
		}
		if isRPSAction(l)(readerOptions, mode, RPSActionModeContinue) {
			l.Debugf("Character [%d] chose to continue their RPS session.", s.CharacterId())
			if err := emitRPSContinueFunc(l, ctx, s.CharacterId(), s.WorldId(), s.ChannelId()); err != nil {
				l.WithError(err).Errorf("Unable to emit RPS CONTINUE command for character [%d].", s.CharacterId())
			}
			return
		}
		if isRPSAction(l)(readerOptions, mode, RPSActionModeExit) {
			l.Debugf("Character [%d] exited their RPS session.", s.CharacterId())
			if err := emitRPSCollectFunc(l, ctx, s.CharacterId(), s.WorldId(), s.ChannelId()); err != nil {
				l.WithError(err).Errorf("Unable to emit RPS COLLECT command for character [%d].", s.CharacterId())
			}
			return
		}
		if isRPSAction(l)(readerOptions, mode, RPSActionModeStart) {
			l.Debugf("Character [%d] issued RPS START; opening the first round.", s.CharacterId())
			if err := emitRPSBeginFunc(l, ctx, s.CharacterId(), s.WorldId(), s.ChannelId()); err != nil {
				l.WithError(err).Errorf("Unable to emit RPS BEGIN command for character [%d].", s.CharacterId())
			}
			return
		}
		if isRPSAction(l)(readerOptions, mode, RPSActionModeUpdate) {
			l.Debugf("Character [%d] issued RPS UPDATE/timeout; no-op, TTL sweeper reaps abandoned sessions.", s.CharacterId())
			return
		}
		if isRPSAction(l)(readerOptions, mode, RPSActionModeRetry) {
			l.Debugf("Character [%d] chose to retry (restart) their RPS session.", s.CharacterId())
			if err := emitRPSRetryFunc(l, ctx, s.CharacterId(), s.WorldId(), s.ChannelId()); err != nil {
				l.WithError(err).Errorf("Unable to emit RPS RETRY command for character [%d].", s.CharacterId())
			}
			return
		}
		l.Warnf("Character [%d] issued an unhandled RPS_ACTION mode [%d].", s.CharacterId(), mode)
	}
}

// isRPSAction mirrors isStorageOperation (storage_operation.go): it resolves
// the incoming sub-op byte's symbolic name via the tenant `operations` table
// and reports whether it matches the given key.
func isRPSAction(l logrus.FieldLogger) func(options map[string]interface{}, op byte, key RPSActionMode) bool {
	return func(options map[string]interface{}, op byte, key RPSActionMode) bool {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		res, ok := codes[string(key)].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}
		return byte(res) == op
	}
}
