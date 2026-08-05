package report

import (
	"atlas-ban/character"
	"atlas-ban/chat"
	"atlas-ban/kafka/message"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	report2 "atlas-ban/kafka/message/report"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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
