package chair

import (
	"atlas-chairs/kafka/message"
	character2 "atlas-chairs/kafka/message/character"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func setupProcessorTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func TestGetById_Success(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	characterId := uint32(12345)
	chairId := uint32(1)
	chairType := "FIXED"

	// Set up registry directly
	GetRegistry().Set(tctx, characterId, Model{id: chairId, chairType: chairType})

	// Test GetById
	p := NewProcessor(l, tctx)
	m, err := p.GetById(characterId)
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}

	if m.Id() != chairId {
		t.Errorf("Expected chair id %d, got %d", chairId, m.Id())
	}

	if m.Type() != chairType {
		t.Errorf("Expected chair type %s, got %s", chairType, m.Type())
	}
}

func TestGetById_NotFound(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	nonExistentCharacter := uint32(99999)

	p := NewProcessor(l, tctx)
	_, err := p.GetById(nonExistentCharacter)

	if err == nil {
		t.Fatal("Expected error for non-existent character, got nil")
	}
}

func TestGetById_MultipleCharacters(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	// Set up multiple chairs
	chars := []struct {
		characterId uint32
		chairId     uint32
		chairType   string
	}{
		{100, 0, "FIXED"},
		{200, 3010001, "PORTABLE"},
		{300, 2, "FIXED"},
	}

	for _, c := range chars {
		GetRegistry().Set(tctx, c.characterId, Model{id: c.chairId, chairType: c.chairType})
	}

	p := NewProcessor(l, tctx)

	// Verify each character's chair
	for _, c := range chars {
		m, err := p.GetById(c.characterId)
		if err != nil {
			t.Errorf("GetById(%d) failed: %v", c.characterId, err)
			continue
		}
		if m.Id() != c.chairId {
			t.Errorf("Character %d: expected chair id %d, got %d", c.characterId, c.chairId, m.Id())
		}
		if m.Type() != c.chairType {
			t.Errorf("Character %d: expected chair type %s, got %s", c.characterId, c.chairType, m.Type())
		}
	}
}

func TestGetById_AfterClear(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	characterId := uint32(12345)

	// Set up then clear
	GetRegistry().Set(tctx, characterId, Model{id: 1, chairType: "FIXED"})
	GetRegistry().Clear(tctx, characterId)

	p := NewProcessor(l, tctx)
	_, err := p.GetById(characterId)

	if err == nil {
		t.Fatal("Expected error after clear, got nil")
	}
}

func TestModel_Accessors(t *testing.T) {
	chairId := uint32(42)
	chairType := "PORTABLE"

	m := Model{id: chairId, chairType: chairType}

	if m.Id() != chairId {
		t.Errorf("Id() expected %d, got %d", chairId, m.Id())
	}

	if m.Type() != chairType {
		t.Errorf("Type() expected %s, got %s", chairType, m.Type())
	}
}

func TestModel_FixedChairTypes(t *testing.T) {
	testCases := []struct {
		name      string
		id        uint32
		chairType string
	}{
		{"Fixed chair 0", 0, "FIXED"},
		{"Fixed chair 1", 1, "FIXED"},
		{"Fixed chair 10", 10, "FIXED"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{id: tc.id, chairType: tc.chairType}
			if m.Id() != tc.id {
				t.Errorf("Expected id %d, got %d", tc.id, m.Id())
			}
			if m.Type() != tc.chairType {
				t.Errorf("Expected type %s, got %s", tc.chairType, m.Type())
			}
		})
	}
}

func TestModel_PortableChairTypes(t *testing.T) {
	// Portable chairs have item IDs in the 301xxxx range
	testCases := []struct {
		name      string
		id        uint32
		chairType string
	}{
		{"Portable chair 3010000", 3010000, "PORTABLE"},
		{"Portable chair 3010001", 3010001, "PORTABLE"},
		{"Portable chair 3019999", 3019999, "PORTABLE"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{id: tc.id, chairType: tc.chairType}
			if m.Id() != tc.id {
				t.Errorf("Expected id %d, got %d", tc.id, m.Id())
			}
			if m.Type() != tc.chairType {
				t.Errorf("Expected type %s, got %s", tc.chairType, m.Type())
			}
		})
	}
}

func recoveryTestContext(t *testing.T, recoveryHP uint32, recoveryMP uint32) (context.Context, func() int) {
	t.Helper()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		id := path.Base(r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"setups","id":"%s","attributes":{"recoveryHP":%d,"recoveryMP":%d}}}`, id, recoveryHP, recoveryMP)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")
	return tenant.WithContext(context.Background(), testTenant()), func() int { return calls }
}

func recoveryMessages(t *testing.T, buf *message.Buffer) []character2.Command[json.RawMessage] {
	t.Helper()
	var out []character2.Command[json.RawMessage]
	for _, m := range buf.GetAll()[character2.EnvCommandTopic] {
		var c character2.Command[json.RawMessage]
		if err := json.Unmarshal(m.Value, &c); err != nil {
			t.Fatalf("unmarshal emitted command: %v", err)
		}
		out = append(out, c)
	}
	return out
}

func amountOf(t *testing.T, raw json.RawMessage) int16 {
	t.Helper()
	var b struct {
		Amount int16 `json:"amount"`
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	return b.Amount
}

func TestRecover_SeatedRecoveryChair_AppliesItemValues(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, calls := recoveryTestContext(t, 60, 60) // 03010136-style both-stats chair
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1001)
	GetRegistry().Set(tctx, characterId, Model{id: 3010136, chairType: "PORTABLE"})

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 60, 60); err != nil {
		t.Fatalf("Recover: %v", err)
	}

	msgs := recoveryMessages(t, buf)
	if len(msgs) != 2 {
		t.Fatalf("expected CHANGE_HP+CHANGE_MP, got %d messages", len(msgs))
	}
	if msgs[0].Type != character2.CommandChangeHP || amountOf(t, msgs[0].Body) != 60 {
		t.Errorf("first message: got %s/%d, want CHANGE_HP/60", msgs[0].Type, amountOf(t, msgs[0].Body))
	}
	if msgs[1].Type != character2.CommandChangeMP || amountOf(t, msgs[1].Body) != 60 {
		t.Errorf("second message: got %s/%d, want CHANGE_MP/60", msgs[1].Type, amountOf(t, msgs[1].Body))
	}
	if calls() != 1 {
		t.Errorf("expected 1 setup-data lookup, got %d", calls())
	}
	m, _ := GetRegistry().Get(tctx, characterId)
	if m.LastHpRecoveryAt() == 0 || m.LastMpRecoveryAt() == 0 {
		t.Error("expected recovery timestamps to be recorded")
	}
}

func TestRecover_SeatedRecoveryChair_ItemValueOverridesClaim(t *testing.T) {
	// HP-only chair (recoveryHP=50, recoveryMP=0); claim lies with hp=30000, mp=5.
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, _ := recoveryTestContext(t, 50, 0)
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1002)
	GetRegistry().Set(tctx, characterId, Model{id: 3010000, chairType: "PORTABLE"})

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 30000, 5); err != nil {
		t.Fatalf("Recover: %v", err)
	}

	msgs := recoveryMessages(t, buf)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	// HP: item value 50 applied, forged 30000 ignored.
	if msgs[0].Type != character2.CommandChangeHP || amountOf(t, msgs[0].Body) != 50 {
		t.Errorf("HP: got %s/%d, want CHANGE_HP/50", msgs[0].Type, amountOf(t, msgs[0].Body))
	}
	// MP: chair doesn't cover it -> natural pass-through of the claim.
	if msgs[1].Type != character2.CommandChangeMP || amountOf(t, msgs[1].Body) != 5 {
		t.Errorf("MP: got %s/%d, want CHANGE_MP/5", msgs[1].Type, amountOf(t, msgs[1].Body))
	}
}

func TestRecover_RateLimited_Drops(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, _ := recoveryTestContext(t, 50, 0)
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1003)
	now := time.Now().UnixMilli()
	GetRegistry().Set(tctx, characterId, Model{id: 3010000, chairType: "PORTABLE"}.WithHpRecoveryAt(now))

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 50, 0); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(recoveryMessages(t, buf)) != 0 {
		t.Error("expected rate-limited tick to emit nothing")
	}
	m, _ := GetRegistry().Get(tctx, characterId)
	if m.LastHpRecoveryAt() != now {
		t.Error("expected timestamp unchanged on rejected tick")
	}
}

func TestRecover_NotSeated_PassThrough(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, calls := recoveryTestContext(t, 50, 0)
	f := field.NewBuilder(0, 1, 100000000).Build()

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, uint32(1004), 17, -3); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	msgs := recoveryMessages(t, buf)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 pass-through messages, got %d", len(msgs))
	}
	if amountOf(t, msgs[0].Body) != 17 {
		t.Errorf("HP pass-through: got %d, want 17", amountOf(t, msgs[0].Body))
	}
	// Negative claims (jms clamp-to-max corrections) pass through unchanged.
	if amountOf(t, msgs[1].Body) != -3 {
		t.Errorf("MP pass-through: got %d, want -3", amountOf(t, msgs[1].Body))
	}
	if calls() != 0 {
		t.Errorf("not-seated tick must not hit setup data, got %d calls", calls())
	}
}

func TestRecover_FixedSeat_PassThrough(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, calls := recoveryTestContext(t, 50, 0)
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1005)
	GetRegistry().Set(tctx, characterId, Model{id: 2, chairType: "FIXED"})

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 25, 0); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	msgs := recoveryMessages(t, buf)
	if len(msgs) != 1 || amountOf(t, msgs[0].Body) != 25 {
		t.Fatalf("expected single HP pass-through of 25, got %d messages", len(msgs))
	}
	if calls() != 0 {
		t.Errorf("fixed-seat tick must not hit setup data, got %d calls", calls())
	}
}

func TestRecover_NonRecoveryPortable_PassThrough(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, _ := recoveryTestContext(t, 0, 0) // portable chair with no recovery stats
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1006)
	GetRegistry().Set(tctx, characterId, Model{id: 3010900, chairType: "PORTABLE"})

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 17, 3); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	msgs := recoveryMessages(t, buf)
	if len(msgs) != 2 || amountOf(t, msgs[0].Body) != 17 || amountOf(t, msgs[1].Body) != 3 {
		t.Fatalf("expected claimed 17/3 pass-through, got %d messages", len(msgs))
	}
}

func TestRecover_DataLookupFailure_DropsSeatedTick(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")
	tctx := tenant.WithContext(context.Background(), testTenant())
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1007)
	GetRegistry().Set(tctx, characterId, Model{id: 3010000, chairType: "PORTABLE"})

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 50, 0); err != nil {
		t.Fatalf("Recover must swallow the lookup failure, got: %v", err)
	}
	if len(recoveryMessages(t, buf)) != 0 {
		t.Error("expected fail-closed drop (never fall back to claimed value)")
	}
}

func TestRecover_ZeroClaims_EmitNothing(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, _ := recoveryTestContext(t, 0, 0)
	f := field.NewBuilder(0, 1, 100000000).Build()

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, uint32(1008), 0, 0); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(recoveryMessages(t, buf)) != 0 {
		t.Error("zero claims must emit nothing")
	}
}

func TestRecover_ClearResetsTimestamps(t *testing.T) {
	// Standing up removes the registration (and its timestamps) entirely;
	// a subsequent tick takes the not-seated pass-through branch.
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, _ := recoveryTestContext(t, 50, 0)
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1009)
	GetRegistry().Set(tctx, characterId, Model{id: 3010000, chairType: "PORTABLE"}.WithHpRecoveryAt(time.Now().UnixMilli()))

	// Registry-level Clear used directly: the processor Clear emits a
	// CANCELLED status event via producer.ProviderImpl, which dials
	// BOOTSTRAP_SERVERS and errors out with no broker present in this test
	// environment. This test only depends on the registry-removal effect
	// (FR-4.5), so GetRegistry().Clear is used instead of NewProcessor(...).Clear.
	GetRegistry().Clear(tctx, characterId)

	if _, ok := GetRegistry().Get(tctx, characterId); ok {
		t.Fatal("expected registration (and timestamps) gone after Clear")
	}

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 17, 0); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	msgs := recoveryMessages(t, buf)
	if len(msgs) != 1 || amountOf(t, msgs[0].Body) != 17 {
		t.Fatal("expected not-seated pass-through after stand-up")
	}
}
