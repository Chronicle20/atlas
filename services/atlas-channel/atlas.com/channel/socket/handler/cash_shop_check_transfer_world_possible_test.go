package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func transferWorldPossiblePacket(l logrus.FieldLogger, ctx context.Context, characterId uint32, birthDate uint32, spw string) *request.Reader {
	p := cashsb.NewCheckTransferWorldPossible(characterId, birthDate, spw)
	raw := p.Encode(l, ctx)(nil)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)
	return &reader
}

// FR requirement: on v95+ the credential is validated against the account
// PIC.
func TestTransferWorldPossibleV95ValidatesAgainstPic(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
	env.withAccount(buildAccount("s3cr3t", 0))
	env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "s3cr3t"))

	if got := env.lastAnnouncedResultByte(); got != 0x20 {
		t.Fatalf("result byte = 0x%02X, want ALLOWED 0x20", got)
	}
}

// task-227 Task 26 ruling 2/3: jms_v185 sends the credential as a string
// (like v95+) AND drops the characterId field entirely; the handler must
// fall back to the session's own character id.
func TestTransferWorldPossibleJmsUsesSessionCharacterIdAndStringCredential(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "JMS", 185, 1)
	env.withAccount(buildAccount("s3cr3t", 0))
	// jms body has no characterId field on the wire at all -- pass 0, the
	// codec's own Encode will not emit it.
	env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, 0, 0, "s3cr3t"))

	if got := env.lastAnnouncedResultByte(); got != 0x20 {
		t.Fatalf("result byte = 0x%02X, want ALLOWED 0x20", got)
	}
	if len(env.announced) == 0 {
		t.Fatal("nothing was announced")
	}
	b := env.announced[len(env.announced)-1].body
	// jms body has no birthDate field either (checkTransferWorldPossibleResultHasBirthDate
	// is false for JMS), so the announced body is characterId(4) + result(1) +
	// hasWorldList(1) only.
	if len(b) < 4 {
		t.Fatalf("announced body length %d, too short to carry characterId", len(b))
	}
	gotCharacterId := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	if gotCharacterId != checkPossibleTestCharacterId {
		t.Fatalf("announced characterId = %d, want the session's own character id %d", gotCharacterId, checkPossibleTestCharacterId)
	}
}

// task-227 Task 26 ruling 3: pre-v95 GMS validates against the account's
// stored BirthDate.
func TestTransferWorldPossiblePreV95ValidatesAgainstStoredBirthDate(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 83, 1)
	env.withAccount(buildAccount("unrelated-pic", 19900101))
	env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 19900101, ""))

	if got := env.lastAnnouncedResultByte(); got != 0x20 {
		t.Fatalf("result byte = 0x%02X, want ALLOWED 0x20", got)
	}
}

// An unset (0) stored birth date always fails, matching the sibling
// name-change handler's behaviour (task-227 Task 26 ruling 3).
func TestTransferWorldPossibleUnsetStoredBirthDateAlwaysFails(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 83, 1)
	env.withAccount(buildAccount("", 0))
	env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, ""))

	if got := env.lastAnnouncedResultByte(); got == 0x20 {
		t.Fatal("an unset (0) stored birth date must never pass the check")
	}
}

// The credential must be validated before the check is answered, and the
// attempt (failure) must be recorded.
func TestTransferWorldPossibleValidatesBeforeAnswering(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
	env.withAccount(buildAccount("correct", 0))
	env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "wrong"))

	if got := env.lastAnnouncedResultByte(); got == 0x20 {
		t.Fatal("a wrong credential must not be answered ALLOWED")
	}
	if len(env.picAttempts) != 1 || env.picAttempts[0] != false {
		t.Fatalf("pic attempts = %v, want exactly one failed attempt", env.picAttempts)
	}
}

// A lockout on this op has no dedicated arm, so it reuses UNKNOWN_ERROR --
// task-227 Task 26 ruling 4's "reuse the existing rejection path".
func TestTransferWorldPossibleLockoutReusesUnknownError(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
	env.withAccount(buildAccount("correct", 0))
	env.limitReached = true
	env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "wrong"))

	if got := env.lastAnnouncedResultByte(); got != 0x2F {
		t.Fatalf("result byte = 0x%02X, want UNKNOWN_ERROR 0x2F", got)
	}
}

// The credential must never reach a log line.
func TestTransferWorldPossibleNeverLogsTheCredential(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
	env.withAccount(buildAccount("s3cr3tSPW", 0))
	env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "s3cr3tSPW"))

	if strings.Contains(env.logOutput(), "s3cr3tSPW") {
		t.Fatal("the SPW credential leaked into the logs")
	}
}
