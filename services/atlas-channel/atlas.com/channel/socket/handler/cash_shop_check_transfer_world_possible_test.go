package handler

import (
	"atlas-channel/character"
	channelworld "atlas-channel/world"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
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

// FR-4.7: when the transferring character is the account's LAST character
// in the source world, the check must ALSO write a pink-text storage
// warning -- and the result must still be answered ALLOWED (the warning is
// advisory, never a gate). This is the original brief's
// TestWorldTransferCheckWarnsWhenStrandingStorage, asserting both halves.
func TestWorldTransferCheckWarnsWhenStrandingStorage(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
	env.withAccount(buildAccount("s3cr3t", 0))
	env.withCharactersInWorld(character.NewModelBuilder().SetId(checkPossibleTestCharacterId).MustBuild())
	env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "s3cr3t"))

	if got := env.lastAnnouncedResultByte(); got != 0x20 {
		t.Fatalf("result byte = 0x%02X, want ALLOWED 0x20", got)
	}
	if !env.storageWarningWasAnnounced() {
		t.Fatal("expected the storage-stranding pink-text warning to be written")
	}
}

// FR-4.7: when the account has ANOTHER character left behind in the source
// world, storage is not stranded (it stays reachable through the sibling),
// so no warning is due -- and the result is unaffected.
func TestWorldTransferCheckNoWarningWhenAnotherCharacterRemains(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
	env.withAccount(buildAccount("s3cr3t", 0))
	env.withCharactersInWorld(
		character.NewModelBuilder().SetId(checkPossibleTestCharacterId).MustBuild(),
		character.NewModelBuilder().SetId(checkPossibleTestCharacterId+1).MustBuild(),
	)
	env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "s3cr3t"))

	if got := env.lastAnnouncedResultByte(); got != 0x20 {
		t.Fatalf("result byte = 0x%02X, want ALLOWED 0x20", got)
	}
	if env.storageWarningWasAnnounced() {
		t.Fatal("no warning is due when another character remains in the source world")
	}
}

// FR-4.7 ruling 2: the warning FAILS OPEN. A failed last-character lookup
// must never reject the check or fail the handler -- it must simply skip
// the courtesy warning and answer normally.
func TestWorldTransferCheckLookupErrorFailsOpen(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
	env.withAccount(buildAccount("s3cr3t", 0))
	env.withCharactersInWorldErr(errors.New("atlas-character unavailable"))
	env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "s3cr3t"))

	if got := env.lastAnnouncedResultByte(); got != 0x20 {
		t.Fatalf("result byte = 0x%02X, want ALLOWED 0x20 (a courtesy lookup failure must not block a legitimate transfer)", got)
	}
	if env.storageWarningWasAnnounced() {
		t.Fatal("a failed lookup must not emit the warning")
	}
}

// decodeAllowedWorldNames pulls the world-name list out of a GMS pre-v95
// CheckTransferWorldPossibleResult body: characterId(4) + result(1) +
// birthDate(4) + hasWorldList(1) [+ count(4) + count x (len(2) + bytes)].
func decodeAllowedWorldNames(t *testing.T, b []byte) []string {
	t.Helper()
	if len(b) < 10 {
		t.Fatalf("announced body length %d, too short to carry hasWorldList", len(b))
	}
	if b[9] == 0 {
		return nil
	}
	i := 10
	rd32 := func() int {
		v := int(b[i]) | int(b[i+1])<<8 | int(b[i+2])<<16 | int(b[i+3])<<24
		i += 4
		return v
	}
	count := rd32()
	names := make([]string, 0, count)
	for n := 0; n < count; n++ {
		if i+2 > len(b) {
			t.Fatalf("body truncated reading name %d of %d", n, count)
		}
		length := int(b[i]) | int(b[i+1])<<8
		i += 2
		if i+length > len(b) {
			t.Fatalf("body truncated reading name %d of %d", n, count)
		}
		names = append(names, string(b[i:i+length]))
		i += length
	}
	return names
}

func (e *checkPossibleHandlerEnv) lastAnnouncedResultBody() []byte {
	e.t.Helper()
	for i := len(e.announced) - 1; i >= 0; i-- {
		if e.announced[i].writer == chatpkt.WorldMessageWriter {
			continue
		}
		return e.announced[i].body
	}
	e.t.Fatal("no check result was announced")
	return nil
}

// The crash this fixes: CUITransferWorldSelectDlg::OnCreate (v83 @0x7efc22)
// calls CCtrlComboBox::SetSelected(0) on the combo it just filled, and
// SetSelected (@0x4c738b -> sub_4C7379 @0x4c7379) dereferences the null head
// of an empty ZList. An ALLOWED arm therefore MUST carry a non-empty list.
func TestTransferWorldPossibleAllowedCarriesTheWorldNameList(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 83, 1)
	env.withAccount(buildAccount("", 19900101))
	env.withWorlds(buildWorld(0, "Scania"), buildWorld(1, "Bera"))
	env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 19900101, ""))

	if got := env.lastAnnouncedResultByte(); got != 0x20 {
		t.Fatalf("result byte = 0x%02X, want ALLOWED 0x20", got)
	}
	got := decodeAllowedWorldNames(t, env.lastAnnouncedResultBody())
	want := []string{"Scania", "Bera"}
	if len(got) != len(want) {
		t.Fatalf("world names = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("world names = %v, want %v", got, want)
		}
	}
}

// The client sends back the combo INDEX as nTargetWorld (sub_7F00D2
// @0x7f00d2 stores m_nSelected; GetResult @0x7f00e2 returns it verbatim;
// OnButtonClicked @0x7ef6e3 passes it to SendBuyTransferWorldItemPacket),
// and handleBuyWorldTransfer reads it as a world id. So the list must be
// indexed by world id — a world set arriving out of order, or with a gap,
// must not shift any world off its own index.
func TestTransferWorldNameListIsIndexedByWorldId(t *testing.T) {
	l := logrus.New()
	l.SetOutput(io.Discard)

	t.Run("out of order", func(t *testing.T) {
		got := transferWorldNameList(l, []channelworld.Model{buildWorld(2, "Broa"), buildWorld(0, "Scania"), buildWorld(1, "Bera")})
		want := []string{"Scania", "Bera", "Broa"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("names = %v, want %v", got, want)
		}
	})

	t.Run("gap keeps later worlds on their own index", func(t *testing.T) {
		got := transferWorldNameList(l, []channelworld.Model{buildWorld(0, "Scania"), buildWorld(2, "Broa")})
		want := []string{"Scania", "", "Broa"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("names = %v, want %v", got, want)
		}
	})

	t.Run("empty world set yields no list", func(t *testing.T) {
		if got := transferWorldNameList(l, nil); got != nil {
			t.Fatalf("names = %v, want nil", got)
		}
	})
}

// A failed world-list lookup must NOT answer ALLOWED — an ALLOWED arm with an
// empty list crashes the client, so a refusal is the safe answer.
func TestTransferWorldPossibleWorldListFailureRefusesRatherThanCrashing(t *testing.T) {
	t.Run("lookup error", func(t *testing.T) {
		env := newCheckPossibleHandlerEnv(t, "GMS", 83, 1)
		env.withAccount(buildAccount("", 19900101))
		env.withWorldsErr(errors.New("atlas-world unavailable"))
		env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 19900101, ""))

		if got := env.lastAnnouncedResultByte(); got != 0x2F {
			t.Fatalf("result byte = 0x%02X, want UNKNOWN_ERROR 0x2F", got)
		}
	})

	t.Run("empty world set", func(t *testing.T) {
		env := newCheckPossibleHandlerEnv(t, "GMS", 83, 1)
		env.withAccount(buildAccount("", 19900101))
		env.withWorlds()
		env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 19900101, ""))

		if got := env.lastAnnouncedResultByte(); got != 0x2F {
			t.Fatalf("result byte = 0x%02X, want UNKNOWN_ERROR 0x2F", got)
		}
	})
}

// The FR-4.7 warning goes to a client sitting in the Cash Shop, which has no
// status bar. PINK_TEXT (arm 5) reaches CHATLOG_ADD @0x4906b5, whose body is
// entirely inside `if (TSingleton<CUIStatusBar>::ms_pInstance)` — a silent
// no-op there. It must be sent as POP_UP (arm 1), which
// CWvsContext::OnBroadcastMsg @0xa22785 routes to CUtilDlg::Notice with no
// status-bar guard.
func TestWorldTransferStorageWarningUsesPopUpNotPinkText(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
	env.withAccount(buildAccount("s3cr3t", 0))
	env.withCharactersInWorld(character.NewModelBuilder().SetId(checkPossibleTestCharacterId).MustBuild())
	env.handleWorldTransfer(transferWorldPossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "s3cr3t"))

	if got := env.storageWarningModeByte(); got != 0x01 {
		t.Fatalf("storage warning mode byte = 0x%02X, want POP_UP 0x01 (0x05 is PINK_TEXT, which the Cash Shop silently drops)", got)
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
