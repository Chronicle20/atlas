package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	parcelsb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// DueyActionMode names a DUEY_ACTION arm. Values match the config keys in
// docs/packets/dispatchers/duey_action.yaml — SEND is wired by Task 17;
// RECEIVE, DISCARD and CLOSE are wired by this file (Task 18).
type DueyActionMode string

const (
	DueyActionModeSend    DueyActionMode = "SEND"
	DueyActionModeReceive DueyActionMode = "RECEIVE"
	DueyActionModeDiscard DueyActionMode = "DISCARD"
	DueyActionModeClose   DueyActionMode = "CLOSE"
)

// DueyActionHandleFunc is the DUEY_ACTION mode-byte dispatcher, mirroring
// StorageOperationHandleFunc's shape exactly: decode the mode byte, then
// compare it against each arm's CONFIG-RESOLVED code (isDueyAction), never
// a literal — the mode value differs by client version (jms_v185 is
// shifted +1 from every GMS build).
func DueyActionHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := parcelsb.Action{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		mode := p.Mode()
		if isDueyAction(l)(readerOptions, mode, DueyActionModeSend) {
			sp := &parcelsb.ActionSend{}
			sp.Decode(l, ctx)(r, readerOptions)
			handleDueyActionSend(l, ctx, wp)(s, sp)
			return
		}
		if isDueyAction(l)(readerOptions, mode, DueyActionModeReceive) {
			sp := &parcelsb.ActionReceive{}
			sp.Decode(l, ctx)(r, readerOptions)
			handleDueyActionReceive(l, ctx, wp)(s, sp)
			return
		}
		if isDueyAction(l)(readerOptions, mode, DueyActionModeDiscard) {
			sp := &parcelsb.ActionDiscard{}
			sp.Decode(l, ctx)(r, readerOptions)
			handleDueyActionDiscard(l, ctx, wp)(s, sp)
			return
		}
		if isDueyAction(l)(readerOptions, mode, DueyActionModeClose) {
			sp := &parcelsb.ActionClose{}
			sp.Decode(l, ctx)(r, readerOptions)
			handleDueyActionClose(l, ctx, wp)(s, sp)
			return
		}
		l.Warnf("Character [%d] issued an unhandled DUEY_ACTION mode [%d].", s.CharacterId(), mode)
	}
}

func isDueyAction(l logrus.FieldLogger) func(options map[string]interface{}, op byte, key DueyActionMode) bool {
	return func(options map[string]interface{}, op byte, key DueyActionMode) bool {
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
