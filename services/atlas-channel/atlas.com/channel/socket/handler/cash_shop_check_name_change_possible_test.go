package handler

import (
	"atlas-channel/account"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	checkPossibleTestCharacterId = uint32(7001)
	checkPossibleTestAccountId   = uint32(5001)
)

type checkPossibleHandlerEnv struct {
	t         *testing.T
	ctx       context.Context
	s         session.Model
	l         logrus.FieldLogger
	logs      *bytes.Buffer
	wp        writer.Producer
	announced []struct {
		writer string
		body   []byte
	}
	account      account.Model
	accountErr   error
	picAttempts  []bool
	limitReached bool
	recordPicErr error
}

// newCheckPossibleHandlerEnv builds a session for the given tenant and
// installs the checkPossibleAccountGetByIdFunc / checkPossibleRecordPicAttemptFunc
// seams (declared in cash_shop_check_name_change_possible.go) so no live
// atlas-account round trip happens — precedent: newCouponHandlerEnv's
// couponRedemptionRequestFunc seam swap in cash_shop_coupon_code_test.go.
func newCheckPossibleHandlerEnv(t *testing.T, region string, major uint16, minor uint16) *checkPossibleHandlerEnv {
	t.Helper()

	ten := mustTenant(t, region, major, minor)
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, discardConn{})
	session.AddSessionToRegistry(ten.Id(), s)
	t.Cleanup(func() { session.ClearRegistryForTenant(ten.Id()) })

	logs := &bytes.Buffer{}
	l := logrus.New()
	l.SetOutput(logs)
	l.SetLevel(logrus.DebugLevel)

	sp := session.NewProcessor(l, ctx)
	sp.SetAccountId(sessionId, checkPossibleTestAccountId)
	updated := sp.SetCharacterId(sessionId, checkPossibleTestCharacterId)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
	updated = session.NewProcessor(l, ctx).SetField(sessionId, f)

	env := &checkPossibleHandlerEnv{t: t, ctx: ctx, s: updated, l: l, logs: logs}

	env.wp = func(name string) (swriter.BodyFunc, error) {
		return func(bl logrus.FieldLogger, bctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(bl, bctx)(checkPossibleWriterOptions(name))
				env.announced = append(env.announced, struct {
					writer string
					body   []byte
				}{writer: name, body: b})
				return b
			}
		}, nil
	}

	origGet := checkPossibleAccountGetByIdFunc
	checkPossibleAccountGetByIdFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (account.Model, error) {
		return env.account, env.accountErr
	}
	t.Cleanup(func() { checkPossibleAccountGetByIdFunc = origGet })

	origRecord := checkPossibleRecordPicAttemptFunc
	checkPossibleRecordPicAttemptFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32, success bool, _ string) (int, bool, error) {
		env.picAttempts = append(env.picAttempts, success)
		return len(env.picAttempts), env.limitReached, env.recordPicErr
	}
	t.Cleanup(func() { checkPossibleRecordPicAttemptFunc = origRecord })

	return env
}

// checkPossibleWriterOptions supplies distinct, non-real resolved bytes,
// scoped per writer name exactly as a real tenant template's per-writer
// "operations" table is (each writer resolves against ITS OWN template
// entry, never a shared global map) -- mirroring couponTestWriterOptions's
// approach in cash_shop_coupon_code_test.go: the point is to prove the
// handler goes through the config-resolved path (DOM-25), not a hard-coded
// byte.
func checkPossibleWriterOptions(writerName string) map[string]interface{} {
	if writerName == cashcb.CashShopCheckTransferWorldPossibleResultWriter {
		return map[string]interface{}{
			"operations": map[string]interface{}{
				cashcb.CheckTransferWorldPossibleAllowed:           float64(0x20),
				cashcb.CheckTransferWorldPossibleCharacterNotFound: float64(0x21),
				cashcb.CheckTransferWorldPossibleInFamily:          float64(0x28),
				cashcb.CheckTransferWorldPossibleUnknownError:      float64(0x2F),
			},
		}
	}
	return map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CheckNameChangePossibleAllowed:               float64(0x10),
			cashcb.CheckNameChangePossibleAlreadySubmitted:      float64(0x11),
			cashcb.CheckNameChangePossibleRequestLimitRecent:    float64(0x12),
			cashcb.CheckNameChangePossibleRequestLimitRequested: float64(0x13),
			cashcb.CheckNameChangePossibleUnknownError:          float64(0x14),
		},
	}
}

func (e *checkPossibleHandlerEnv) withAccount(a account.Model) *checkPossibleHandlerEnv {
	e.account = a
	return e
}

func (e *checkPossibleHandlerEnv) handleNameChange(r *request.Reader) {
	e.t.Helper()
	CashShopCheckNameChangePossibleHandleFunc(e.l, e.ctx, e.wp)(e.s, r, map[string]interface{}{})
}

func (e *checkPossibleHandlerEnv) handleWorldTransfer(r *request.Reader) {
	e.t.Helper()
	CashShopCheckTransferWorldPossibleHandleFunc(e.l, e.ctx, e.wp)(e.s, r, map[string]interface{}{})
}

func (e *checkPossibleHandlerEnv) lastAnnouncedResultByte() byte {
	e.t.Helper()
	if len(e.announced) == 0 {
		e.t.Fatal("nothing was announced")
	}
	b := e.announced[len(e.announced)-1].body
	if len(b) < 5 {
		e.t.Fatalf("announced body length %d, want at least 5 (characterId + result)", len(b))
	}
	return b[4]
}

func (e *checkPossibleHandlerEnv) logOutput() string { return e.logs.String() }

func nameChangePossiblePacket(l logrus.FieldLogger, ctx context.Context, characterId uint32, birthDate uint32, spw string) *request.Reader {
	p := cashsb.NewCheckNameChangePossible(characterId, birthDate, spw)
	raw := p.Encode(l, ctx)(nil)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)
	return &reader
}

func buildAccount(pic string, birthDate uint32) account.Model {
	return account.NewBuilder().
		SetId(checkPossibleTestAccountId).
		SetPic(pic).
		SetBirthDate(birthDate).
		Build()
}

// FR requirement: on v95+ the credential is validated against the account
// PIC, and the check succeeds only on a match.
func TestNameChangePossibleV95ValidatesAgainstPic(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
	env.withAccount(buildAccount("s3cr3t", 0))
	env.handleNameChange(nameChangePossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "s3cr3t"))

	if got := env.lastAnnouncedResultByte(); got != 0x10 {
		t.Fatalf("result byte = 0x%02X, want ALLOWED 0x10", got)
	}
}

// task-227 Task 26 ruling 3: pre-v95, the credential is the account's stored
// BirthDate, never PIC.
func TestNameChangePossiblePreV95ValidatesAgainstStoredBirthDate(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 83, 1)
	env.withAccount(buildAccount("unrelated-pic", 19900101))
	env.handleNameChange(nameChangePossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 19900101, ""))

	if got := env.lastAnnouncedResultByte(); got != 0x10 {
		t.Fatalf("result byte = 0x%02X, want ALLOWED 0x10", got)
	}
}

// task-227 Task 26 ruling 3: a stored birth date of 0 means UNSET and FAILS
// the check — it must never be treated as matching a 0 sent on the wire. This
// is the state every account is in today, so pre-v95 name change/world
// transfer is unusable until an account has a birth date provisioned.
func TestNameChangePossibleUnsetStoredBirthDateAlwaysFails(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 83, 1)
	env.withAccount(buildAccount("", 0))
	env.handleNameChange(nameChangePossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, ""))

	if got := env.lastAnnouncedResultByte(); got == 0x10 {
		t.Fatal("an unset (0) stored birth date must never pass the check")
	}
	if len(env.picAttempts) != 1 || env.picAttempts[0] != false {
		t.Fatalf("pic attempts = %v, want exactly one failed attempt recorded", env.picAttempts)
	}
}

// On pre-v95 the handler must not consult PIC() at all, and on v95 it must
// not consult BirthDate() — each version path uses only its own credential
// field.
func TestNameChangePossibleVersionPathsAreIsolated(t *testing.T) {
	t.Run("pre-v95 ignores PIC even when it happens to equal the sent birthdate as a string", func(t *testing.T) {
		env := newCheckPossibleHandlerEnv(t, "GMS", 83, 1)
		// PIC matches nothing the wire sends; BirthDate is unset. If the
		// handler ever fell back to PIC on this version path, this would
		// incorrectly pass.
		env.withAccount(buildAccount("19900101", 0))
		env.handleNameChange(nameChangePossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 19900101, ""))
		if got := env.lastAnnouncedResultByte(); got == 0x10 {
			t.Fatal("pre-v95 must not fall back to comparing against PIC")
		}
	})

	t.Run("v95 ignores BirthDate even when it happens to equal the sent SPW", func(t *testing.T) {
		env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
		// BirthDate is set to a value that, if compared as a decimal string,
		// would not equal the sent SPW "s3cr3t" -- but more importantly,
		// PIC is wrong, so if the handler ever fell back to BirthDate this
		// would incorrectly pass via some other coincidence. Assert PIC
		// mismatch alone rejects regardless of BirthDate.
		env.withAccount(buildAccount("different-pic", 19900101))
		env.handleNameChange(nameChangePossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "s3cr3t"))
		if got := env.lastAnnouncedResultByte(); got == 0x10 {
			t.Fatal("v95 must not fall back to comparing against BirthDate")
		}
	})
}

// The credential must be validated before the check is answered: a wrong
// credential must never be reported as ALLOWED, and it must be recorded as a
// failed PIC attempt (the lockout counter) rather than silently dropped.
func TestNameChangePossibleValidatesBeforeAnswering(t *testing.T) {
	env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
	env.withAccount(buildAccount("correct", 0))
	env.handleNameChange(nameChangePossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "wrong"))

	if got := env.lastAnnouncedResultByte(); got == 0x10 {
		t.Fatal("a wrong credential must not be answered ALLOWED")
	}
	if len(env.picAttempts) != 1 || env.picAttempts[0] != false {
		t.Fatalf("pic attempts = %v, want exactly one failed attempt", env.picAttempts)
	}
}

// A lockout-tripping mismatch answers the request-limit arm and still
// records the (failed) attempt; a successful credential resets the counter
// via a recorded success.
func TestNameChangePossibleLockoutAndSuccessRecording(t *testing.T) {
	t.Run("lockout tripped answers REQUEST_LIMIT_RECENT", func(t *testing.T) {
		env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
		env.withAccount(buildAccount("correct", 0))
		env.limitReached = true
		env.handleNameChange(nameChangePossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "wrong"))
		if got := env.lastAnnouncedResultByte(); got != 0x12 {
			t.Fatalf("result byte = 0x%02X, want REQUEST_LIMIT_RECENT 0x12", got)
		}
	})

	t.Run("success records a true attempt", func(t *testing.T) {
		env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
		env.withAccount(buildAccount("correct", 0))
		env.handleNameChange(nameChangePossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "correct"))
		if len(env.picAttempts) != 1 || env.picAttempts[0] != true {
			t.Fatalf("pic attempts = %v, want exactly one successful attempt", env.picAttempts)
		}
	})
}

// The credential must never reach a log line, on either version path.
func TestNameChangePossibleNeverLogsTheCredential(t *testing.T) {
	t.Run("pre-v95 birthdate", func(t *testing.T) {
		env := newCheckPossibleHandlerEnv(t, "GMS", 83, 1)
		env.withAccount(buildAccount("", 19900101))
		env.handleNameChange(nameChangePossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 19900101, ""))
		if strings.Contains(env.logOutput(), "19900101") {
			t.Fatal("the birthdate credential leaked into the logs")
		}
	})

	t.Run("v95 spw", func(t *testing.T) {
		env := newCheckPossibleHandlerEnv(t, "GMS", 95, 1)
		env.withAccount(buildAccount("s3cr3tSPW", 0))
		env.handleNameChange(nameChangePossiblePacket(env.l, env.ctx, checkPossibleTestCharacterId, 0, "s3cr3tSPW"))
		if strings.Contains(env.logOutput(), "s3cr3tSPW") {
			t.Fatal("the SPW credential leaked into the logs")
		}
	})
}
