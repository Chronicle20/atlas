package report

import (
	"atlas-ban/character"
	"atlas-ban/chat"
	"atlas-ban/kafka/message"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	report2 "atlas-ban/kafka/message/report"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type fakeCharacterProcessor struct {
	byId   map[uint32]character.Model
	byName map[string]character.Model
	err    error
}

func (f *fakeCharacterProcessor) GetById(characterId uint32) (character.Model, error) {
	if f.err != nil {
		return character.Model{}, f.err
	}
	m, ok := f.byId[characterId]
	if !ok {
		return character.Model{}, requests.ErrNotFound
	}
	return m, nil
}

func (f *fakeCharacterProcessor) GetByName(name string) (character.Model, error) {
	if f.err != nil {
		return character.Model{}, f.err
	}
	m, ok := f.byName[name]
	if !ok {
		return character.Model{}, model.ErrEmptySlice
	}
	return m, nil
}

type fakeChatProcessor struct {
	lines []chat.Model
	err   error
}

func (f *fakeChatProcessor) RecentInvolving(_ []uint32) ([]chat.Model, error) {
	return f.lines, f.err
}

func makeCharacter(t *testing.T, id uint32, name string) character.Model {
	t.Helper()
	m, err := character.Extract(character.RestModel{Id: id, Name: name})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return m
}

func TestCreateFromCommandHappyPathClaim(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{
		byId:   map[uint32]character.Model{1: makeCharacter(t, 1, "Reporter")},
		byName: map[string]character.Model{"Accused": makeCharacter(t, 2, "Accused")},
	}
	chatP := &fakeChatProcessor{lines: []chat.Model{}}
	p := NewProcessorWithClients(l, testContext(tm), db, charP, chatP)

	buf := message.NewBuffer()
	err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindClaim, ReporterId: 1, AccusedName: "Accused",
		ReasonType: 3, Description: "harassment", ChatClaim: true, ChatLog: "log",
	})
	if err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}

	reports, err := p.GetByTenant()
	if err != nil {
		t.Fatalf("GetByTenant: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	m := reports[0]
	if m.AccusedId() != 2 || m.AccusedName() != "Accused" || m.ReporterName() != "Reporter" {
		t.Errorf("resolution mismatch: %+v", m)
	}
	if m.ChatLog() == nil || *m.ChatLog() != "log" {
		t.Error("chat log not stored verbatim")
	}
	msgs := buf.GetAll()
	if len(msgs[report2.EnvEventTopicStatus]) != 1 {
		t.Fatalf("expected 1 status event, got %+v", msgs)
	}
}

func TestCreateFromCommandNotFoundPersistsNothing(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{
		byId:   map[uint32]character.Model{1: makeCharacter(t, 1, "Reporter")},
		byName: map[string]character.Model{},
	}
	p := NewProcessorWithClients(l, testContext(tm), db, charP, &fakeChatProcessor{})

	buf := message.NewBuffer()
	err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindClaim, ReporterId: 1, AccusedName: "Ghost",
		ReasonType: 3, Description: "x",
	})
	if err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports, _ := p.GetByTenant()
	if len(reports) != 0 {
		t.Fatalf("expected no persisted report, got %d", len(reports))
	}
	// The error status event must still be buffered so the reporter gets
	// the not-found result packet.
	msgs := buf.GetAll()
	if len(msgs[report2.EnvEventTopicStatus]) != 1 {
		t.Fatalf("expected 1 status event, got %+v", msgs)
	}
}

func TestCreateFromCommandCharacterServiceDownIsInternal(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{err: errors.New("connection refused")}
	p := NewProcessorWithClients(l, testContext(tm), db, charP, &fakeChatProcessor{})

	buf := message.NewBuffer()
	if err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindSue, ReporterId: 1, AccusedId: 2, Description: "x",
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports, _ := p.GetByTenant()
	if len(reports) != 0 {
		t.Fatal("expected no persisted report")
	}
}

// TestCreateFromCommandNotFoundVsInternalErrorCode pins the two distinct
// status-event error codes (NOT_FOUND vs INTERNAL) by decoding the buffered
// event's JSON payload, not just its count — so a swap between the two
// error-mapping branches in CreateFromCommand fails this test. The
// not-found case drives the empty-by-name-list path (model.ErrEmptySlice
// from a zero-filter model.First, per
// libs/atlas-model/model/processor.go:552); the internal case drives a
// character-service transport failure. Asserting both codes AND that they
// differ catches a mapping regression that a count-only assertion (as in
// TestCreateFromCommandNotFoundPersistsNothing /
// TestCreateFromCommandCharacterServiceDownIsInternal) would miss if the
// two outcomes were swapped.
func TestCreateFromCommandNotFoundVsInternalErrorCode(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Not-found: accused name has no entry in the fake's byName map, so
	// GetByName's underlying zero-filter model.First returns
	// model.ErrEmptySlice.
	dbNF := setupTestDatabase(t)
	tmNF := sampleTenant()
	charPNF := &fakeCharacterProcessor{
		byId:   map[uint32]character.Model{1: makeCharacter(t, 1, "Reporter")},
		byName: map[string]character.Model{},
	}
	pNF := NewProcessorWithClients(l, testContext(tmNF), dbNF, charPNF, &fakeChatProcessor{})
	bufNF := message.NewBuffer()
	if err := pNF.CreateFromCommand(bufNF)(report2.CreateCommandBody{
		Kind: report2.KindClaim, ReporterId: 1, AccusedName: "Ghost", Description: "x",
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	nf := decodeStatusEventErrorCode(t, bufNF)

	// Internal: the character service is entirely down (transport error on
	// every call), so the reporter lookup itself fails.
	dbIn := setupTestDatabase(t)
	tmIn := sampleTenant()
	charPIn := &fakeCharacterProcessor{err: errors.New("connection refused")}
	pIn := NewProcessorWithClients(l, testContext(tmIn), dbIn, charPIn, &fakeChatProcessor{})
	bufIn := message.NewBuffer()
	if err := pIn.CreateFromCommand(bufIn)(report2.CreateCommandBody{
		Kind: report2.KindSue, ReporterId: 1, AccusedId: 2, Description: "x",
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	in := decodeStatusEventErrorCode(t, bufIn)

	if nf != report2.ErrorCodeNotFound {
		t.Errorf("expected NOT_FOUND for empty-by-name-list path, got %q", nf)
	}
	if in != report2.ErrorCodeInternal {
		t.Errorf("expected INTERNAL for transport failure, got %q", in)
	}
	if nf == in {
		t.Fatalf("NOT_FOUND and INTERNAL codes must differ: got %q for both", nf)
	}
}

func decodeStatusEventErrorCode(t *testing.T, buf *message.Buffer) string {
	t.Helper()
	msgs := buf.GetAll()
	evs := msgs[report2.EnvEventTopicStatus]
	if len(evs) != 1 {
		t.Fatalf("expected 1 status event, got %d", len(evs))
	}
	var se report2.StatusEvent
	if err := json.Unmarshal(evs[0].Value, &se); err != nil {
		t.Fatalf("decode status event: %v", err)
	}
	if se.Status != report2.EventStatusError {
		t.Fatalf("expected ERROR status, got %q", se.Status)
	}
	return se.ErrorCode
}

// decodeCreatedStatusEvent returns the single buffered CREATED status event,
// failing if the buffer holds an ERROR instead.
func decodeCreatedStatusEvent(t *testing.T, buf *message.Buffer) report2.StatusEvent {
	t.Helper()
	evs := buf.GetAll()[report2.EnvEventTopicStatus]
	if len(evs) != 1 {
		t.Fatalf("expected 1 status event, got %d", len(evs))
	}
	var se report2.StatusEvent
	if err := json.Unmarshal(evs[0].Value, &se); err != nil {
		t.Fatalf("decode status event: %v", err)
	}
	if se.Status != report2.EventStatusCreated {
		t.Fatalf("expected CREATED status, got %q (errorCode %q)", se.Status, se.ErrorCode)
	}
	return se
}

// seedClaims inserts n already-persisted claims for the reporter, aged by
// `age`. CreatedAt is set explicitly because GORM only fills the field when
// it is zero — which is what lets these tests place rows on either side of
// the rolling window boundary.
func seedClaims(t *testing.T, db *gorm.DB, tenantId uuid.UUID, reporterId uint32, n int, age time.Duration) {
	t.Helper()
	at := time.Now().Add(-age)
	for i := 0; i < n; i++ {
		e := Entity{
			Id:          uuid.New(),
			TenantId:    tenantId,
			Kind:        string(KindClaim),
			ReporterId:  reporterId,
			AccusedId:   2,
			AccusedName: "Accused",
			ReasonType:  3,
			Description: "seed",
			Status:      string(StatusOpen),
			CreatedAt:   at,
			UpdatedAt:   at,
		}
		if err := db.Create(&e).Error; err != nil {
			t.Fatalf("seed claim %d: %v", i, err)
		}
	}
}

func quotaTestProcessor(t *testing.T, db *gorm.DB, tm tenant.Model) Processor {
	t.Helper()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{
		// Id 2 is present as well as by-name: sue resolves the accused by id.
		byId: map[uint32]character.Model{
			1: makeCharacter(t, 1, "Reporter"),
			2: makeCharacter(t, 2, "Accused"),
		},
		byName: map[string]character.Model{"Accused": makeCharacter(t, 2, "Accused")},
	}
	return NewProcessorWithClients(l, testContext(tm), db, charP, &fakeChatProcessor{})
}

func claimCommand() report2.CreateCommandBody {
	return report2.CreateCommandBody{
		Kind: report2.KindClaim, ReporterId: 1, AccusedName: "Accused",
		ReasonType: 3, Description: "harassment",
	}
}

// TestCreateFromCommandClaimReportsTrueRemaining pins the count the client
// renders as "you have %d reports left this week". Before quota enforcement
// this was a hard-coded 100 on every claim, which is what play-testing saw.
func TestCreateFromCommandClaimReportsTrueRemaining(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	seedClaims(t, db, tm.Id(), 1, 3, time.Hour)

	buf := message.NewBuffer()
	if err := quotaTestProcessor(t, db, tm).CreateFromCommand(buf)(claimCommand()); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}

	se := decodeCreatedStatusEvent(t, buf)
	if !se.HasRemaining {
		t.Fatal("want hasRemaining=true on a claim")
	}
	want := int32(MaxClaimsPerWindow - 4)
	if se.Remaining != want {
		t.Fatalf("want remaining=%d after 3 prior claims plus this one, got %d", want, se.Remaining)
	}
}

// TestCreateFromCommandClaimAtQuotaIsRejected asserts the cap is enforced,
// not merely displayed: the report is not persisted and the reporter gets the
// QUOTA_EXCEEDED code that maps to the client's "exceeded the number of
// reports available" notice.
func TestCreateFromCommandClaimAtQuotaIsRejected(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	seedClaims(t, db, tm.Id(), 1, MaxClaimsPerWindow, time.Hour)

	p := quotaTestProcessor(t, db, tm)
	buf := message.NewBuffer()
	if err := p.CreateFromCommand(buf)(claimCommand()); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}

	if code := decodeStatusEventErrorCode(t, buf); code != report2.ErrorCodeQuotaExceeded {
		t.Fatalf("want %s, got %s", report2.ErrorCodeQuotaExceeded, code)
	}
	reports, err := p.GetByTenant()
	if err != nil {
		t.Fatalf("GetByTenant: %v", err)
	}
	if len(reports) != MaxClaimsPerWindow {
		t.Fatalf("over-quota claim must not persist: want %d rows, got %d", MaxClaimsPerWindow, len(reports))
	}
}

// TestCreateFromCommandClaimQuotaWindowRolls asserts the window is rolling:
// a full cap's worth of claims older than ClaimQuotaWindow does not block a
// new one.
func TestCreateFromCommandClaimQuotaWindowRolls(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	seedClaims(t, db, tm.Id(), 1, MaxClaimsPerWindow, ClaimQuotaWindow+time.Hour)

	buf := message.NewBuffer()
	if err := quotaTestProcessor(t, db, tm).CreateFromCommand(buf)(claimCommand()); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}

	se := decodeCreatedStatusEvent(t, buf)
	if se.Remaining != int32(MaxClaimsPerWindow-1) {
		t.Fatalf("aged-out claims must not count: want remaining=%d, got %d", MaxClaimsPerWindow-1, se.Remaining)
	}
}

// TestCreateFromCommandSueIgnoresClaimQuota asserts sue is outside the cap and
// carries no quota fields — its result packet has nowhere to put them.
func TestCreateFromCommandSueIgnoresClaimQuota(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	seedClaims(t, db, tm.Id(), 1, MaxClaimsPerWindow, time.Hour)

	buf := message.NewBuffer()
	err := quotaTestProcessor(t, db, tm).CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindSue, ReporterId: 1, AccusedId: 2,
		ReasonType: 1, Description: "cheating",
	})
	if err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}

	se := decodeCreatedStatusEvent(t, buf)
	if se.HasRemaining || se.Remaining != 0 {
		t.Fatalf("sue must carry no quota standing, got hasRemaining=%t remaining=%d", se.HasRemaining, se.Remaining)
	}
}

func TestCreateFromCommandTruncatesOversizedInputs(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{
		byId: map[uint32]character.Model{
			1: makeCharacter(t, 1, "Reporter"),
			2: makeCharacter(t, 2, "Accused"),
		},
	}
	p := NewProcessorWithClients(l, testContext(tm), db, charP, &fakeChatProcessor{})

	buf := message.NewBuffer()
	longDescription := strings.Repeat("d", MaxDescriptionLength+500)
	longLog := strings.Repeat("c", MaxChatLogBytes+500)
	if err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindClaim, ReporterId: 1, AccusedId: 2,
		Description: longDescription, ChatClaim: true, ChatLog: longLog,
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports, _ := p.GetByTenant()
	if len(reports) != 1 {
		t.Fatal("expected persisted report")
	}
	if len(reports[0].Description()) != MaxDescriptionLength {
		t.Errorf("description not capped: %d", len(reports[0].Description()))
	}
	if reports[0].ChatLog() == nil || len(*reports[0].ChatLog()) != MaxChatLogBytes {
		t.Error("chat log not capped")
	}
}

// newTruncationTestProcessor builds a processor against a fresh in-memory DB
// with a resolvable reporter/accused pair, for the truncation-boundary tests
// below.
func newTruncationTestProcessor(t *testing.T) Processor {
	t.Helper()
	db := setupTestDatabase(t)
	tm := sampleTenant()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{
		byId: map[uint32]character.Model{
			1: makeCharacter(t, 1, "Reporter"),
			2: makeCharacter(t, 2, "Accused"),
		},
	}
	return NewProcessorWithClients(l, testContext(tm), db, charP, &fakeChatProcessor{})
}

// TestCreateFromCommandDescriptionTruncationIsRuneSafe uses a 3-byte CJK
// rune so a naive byte-slice at MaxDescriptionLength (a byte count in that
// case) would both keep far fewer than 2000 characters and could split a
// rune in half, producing invalid UTF-8 that Postgres rejects on INSERT.
// The description cap is specified in RUNES, so the stored value must be
// exactly MaxDescriptionLength runes and valid UTF-8.
func TestCreateFromCommandDescriptionTruncationIsRuneSafe(t *testing.T) {
	p := newTruncationTestProcessor(t)
	longDescription := strings.Repeat("가", MaxDescriptionLength+500)

	buf := message.NewBuffer()
	if err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindClaim, ReporterId: 1, AccusedId: 2, Description: longDescription,
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports, _ := p.GetByTenant()
	if len(reports) != 1 {
		t.Fatal("expected persisted report")
	}
	stored := reports[0].Description()
	if !utf8.ValidString(stored) {
		t.Fatalf("truncated description is not valid UTF-8: %q", stored)
	}
	if got := utf8.RuneCountInString(stored); got != MaxDescriptionLength {
		t.Errorf("description not capped by rune count: got %d runes, want %d", got, MaxDescriptionLength)
	}
}

// TestCreateFromCommandChatLogTruncationIsByteCapRuneSafe uses a 3-byte CJK
// rune sized so the MaxChatLogBytes byte boundary lands mid-rune
// (16384 is not a multiple of 3), forcing the truncation to back off to the
// nearest complete rune rather than emit an invalid byte sequence.
func TestCreateFromCommandChatLogTruncationIsByteCapRuneSafe(t *testing.T) {
	p := newTruncationTestProcessor(t)
	// (MaxChatLogBytes/3)+2 copies of a 3-byte rune: comfortably over the
	// byte cap, and 16384 % 3 == 1, so a byte-index cut at exactly
	// MaxChatLogBytes would land one byte into a rune.
	runeCount := MaxChatLogBytes/3 + 2
	longLog := strings.Repeat("가", runeCount)

	buf := message.NewBuffer()
	if err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindSue, ReporterId: 1, AccusedId: 2, Description: "x",
		ChatClaim: true, ChatLog: longLog,
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports, _ := p.GetByTenant()
	if len(reports) != 1 {
		t.Fatal("expected persisted report")
	}
	stored := reports[0].ChatLog()
	if stored == nil {
		t.Fatal("expected chat log to be stored")
	}
	if len(*stored) > MaxChatLogBytes {
		t.Fatalf("chat log exceeds byte cap: %d > %d", len(*stored), MaxChatLogBytes)
	}
	if !utf8.ValidString(*stored) {
		t.Fatalf("truncated chat log is not valid UTF-8: %q", *stored)
	}
	if !strings.HasPrefix(longLog, *stored) {
		t.Fatal("truncated chat log is not a prefix of the input (boundary was split)")
	}
}

// TestCreateFromCommandDescriptionCapBoundary pins the exact-boundary
// behavior: a description of exactly MaxDescriptionLength runes must pass
// through untouched, and MaxDescriptionLength+1 must truncate by exactly
// one rune.
func TestCreateFromCommandDescriptionCapBoundary(t *testing.T) {
	atCap := strings.Repeat("d", MaxDescriptionLength)
	p := newTruncationTestProcessor(t)
	buf := message.NewBuffer()
	if err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindClaim, ReporterId: 1, AccusedId: 2, Description: atCap,
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports, _ := p.GetByTenant()
	if reports[0].Description() != atCap {
		t.Errorf("description at exactly the cap must not be truncated: got %d chars, want %d", utf8.RuneCountInString(reports[0].Description()), MaxDescriptionLength)
	}

	overCap := strings.Repeat("d", MaxDescriptionLength+1)
	p2 := newTruncationTestProcessor(t)
	buf2 := message.NewBuffer()
	if err := p2.CreateFromCommand(buf2)(report2.CreateCommandBody{
		Kind: report2.KindClaim, ReporterId: 1, AccusedId: 2, Description: overCap,
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports2, _ := p2.GetByTenant()
	if got := utf8.RuneCountInString(reports2[0].Description()); got != MaxDescriptionLength {
		t.Errorf("description one over the cap must truncate to exactly the cap: got %d, want %d", got, MaxDescriptionLength)
	}
}

// TestCreateFromCommandChatLogCapBoundary is the chat-log analogue of
// TestCreateFromCommandDescriptionCapBoundary: exactly MaxChatLogBytes bytes
// must pass through untouched, and MaxChatLogBytes+1 must truncate.
func TestCreateFromCommandChatLogCapBoundary(t *testing.T) {
	atCap := strings.Repeat("c", MaxChatLogBytes)
	p := newTruncationTestProcessor(t)
	buf := message.NewBuffer()
	if err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindSue, ReporterId: 1, AccusedId: 2, Description: "x",
		ChatClaim: true, ChatLog: atCap,
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports, _ := p.GetByTenant()
	if reports[0].ChatLog() == nil || *reports[0].ChatLog() != atCap {
		t.Error("chat log at exactly the cap must not be truncated")
	}

	overCap := strings.Repeat("c", MaxChatLogBytes+1)
	p2 := newTruncationTestProcessor(t)
	buf2 := message.NewBuffer()
	if err := p2.CreateFromCommand(buf2)(report2.CreateCommandBody{
		Kind: report2.KindSue, ReporterId: 1, AccusedId: 2, Description: "x",
		ChatClaim: true, ChatLog: overCap,
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports2, _ := p2.GetByTenant()
	if reports2[0].ChatLog() == nil || len(*reports2[0].ChatLog()) != MaxChatLogBytes {
		t.Error("chat log one over the cap must truncate to exactly the cap")
	}
}

func TestCreateFromCommandTranscriptFailureTolerated(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{
		byId: map[uint32]character.Model{
			1: makeCharacter(t, 1, "Reporter"),
			2: makeCharacter(t, 2, "Accused"),
		},
	}
	p := NewProcessorWithClients(l, testContext(tm), db, charP, &fakeChatProcessor{err: errors.New("messages down")})

	buf := message.NewBuffer()
	if err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindSue, ReporterId: 1, AccusedId: 2, Description: "x",
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports, _ := p.GetByTenant()
	if len(reports) != 1 {
		t.Fatal("expected persisted report despite transcript failure")
	}
	if reports[0].ServerTranscript() != nil {
		t.Error("expected nil transcript")
	}
}

func TestUpdateStatusValidationAndNotFound(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{
		byId: map[uint32]character.Model{
			1: makeCharacter(t, 1, "Reporter"),
			2: makeCharacter(t, 2, "Accused"),
		},
	}
	p := NewProcessorWithClients(l, testContext(tm), db, charP, &fakeChatProcessor{})

	buf := message.NewBuffer()
	_ = p.CreateFromCommand(buf)(report2.CreateCommandBody{Kind: report2.KindSue, ReporterId: 1, AccusedId: 2, Description: "x"})
	reports, _ := p.GetByTenant()

	m, err := p.UpdateStatus(reports[0].Id(), StatusActioned)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if m.Status() != StatusActioned {
		t.Errorf("status: got %s", m.Status())
	}

	if _, err = p.UpdateStatus(reports[0].Id(), Status("bogus")); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
	if _, err = p.UpdateStatus(uuid.New(), StatusReviewed); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}
