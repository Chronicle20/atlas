package handler

import (
	"atlas-channel/maker"
	"atlas-channel/socket/writer"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"

	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
)

// Wire fixtures, byte-identical to
// libs/atlas-packet/character/serverbound/maker_skill_test.go. Duplicated
// here (rather than imported) because that package is internal to
// libs/atlas-packet/character/serverbound and this suite exercises the
// decode through the handler's own request.Reader, not the codec directly.
var (
	makerSkillCreateBytes = []byte{
		0x01, 0x00, 0x00, 0x00, // nRecipeClass = 1
		0x92, 0x82, 0x10, 0x00, // nTargetItemID = 1082002
		0x01,                   // bCatalystMounted = true
		0x02, 0x00, 0x00, 0x00, // nNumGemMounted = 2
		0x41, 0x5C, 0x3D, 0x00, // nGemItemID[0] = 4021313 (not held by the character)
		0x42, 0x5C, 0x3D, 0x00, // nGemItemID[1] = 4021314
	}
	makerSkillCreateWithUpgradeBytes = []byte{
		0x02, 0x00, 0x00, 0x00, // nRecipeClass = 2
		0x92, 0x82, 0x10, 0x00, // nTargetItemID = 1082002
		0x00,                   // bCatalystMounted = false
		0x00, 0x00, 0x00, 0x00, // nNumGemMounted = 0
	}
	makerSkillMonsterCrystalBytes = []byte{
		0x03, 0x00, 0x00, 0x00, // nRecipeClass = 3
		0x00, 0x09, 0x3D, 0x00, // nRecipeItemID = 4000000
	}
	makerSkillDisassembleBytes = []byte{
		0x04, 0x00, 0x00, 0x00, // nRecipeClass = 4
		0x92, 0x82, 0x10, 0x00, // nRecipeItemID = 1082002
		0x01, 0x00, 0x00, 0x00, // nTI_DisassembleItem = 1
		0x05, 0x00, 0x00, 0x00, // nSlotPosition_DisassembleItem = 5
	}
)

// makerSkillReader wraps raw wire bytes in a *request.Reader for the handler.
func makerSkillReader(in []byte) *request.Reader {
	req := request.Request(in)
	r := request.NewRequestReader(&req, 0)
	return &r
}

// makerRecorder is a fake writer.Producer that records every announced
// writer name + body, for asserting MAKER_RESULT writes (or their absence).
type makerRecorder struct {
	announced []struct {
		writer string
		body   []byte
	}
}

func (r *makerRecorder) producer() writer.Producer {
	return func(name string) (swriter.BodyFunc, error) {
		return func(l logrus.FieldLogger, ctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(l, ctx)(map[string]interface{}{})
				r.announced = append(r.announced, struct {
					writer string
					body   []byte
				}{writer: name, body: b})
				return b
			}
		}, nil
	}
}

// craftErrDoc is atlas-maker's minimal JSON:API error document shape
// (services/atlas-maker/atlas.com/maker/craft/errors.go's errorDocument).
type craftErrDoc struct {
	Errors []struct {
		Status string `json:"status"`
		Code   string `json:"code"`
		Title  string `json:"title"`
	} `json:"errors"`
}

func writeCraftErrorResponse(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	doc := craftErrDoc{}
	doc.Errors = append(doc.Errors, struct {
		Status string `json:"status"`
		Code   string `json:"code"`
		Title  string `json:"title"`
	}{Status: http.StatusText(status), Code: code, Title: http.StatusText(status)})
	_ = json.NewEncoder(w).Encode(doc)
}

// TestMakerSkillHandlerForwardsEachModeVerbatim pins that the channel does
// not filter the request: every decoded field, including a gem the
// character does not hold, reaches atlas-maker unchanged. Filtering here
// would mask a validation bug that belongs to atlas-maker alone.
func TestMakerSkillHandlerForwardsEachModeVerbatim(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
		check func(t *testing.T, got maker.CraftRequest)
	}{
		{
			name:  "CREATE",
			bytes: makerSkillCreateBytes,
			check: func(t *testing.T, got maker.CraftRequest) {
				if got.Mode != 1 || got.TargetItemId != 1082002 || !got.UseCatalyst {
					t.Fatalf("unexpected create request: %+v", got)
				}
				if len(got.GemItemIds) != 2 || got.GemItemIds[0] != 4021313 || got.GemItemIds[1] != 4021314 {
					t.Fatalf("gem list not forwarded verbatim: %+v", got.GemItemIds)
				}
			},
		},
		{
			name:  "CREATE_WITH_UPGRADE",
			bytes: makerSkillCreateWithUpgradeBytes,
			check: func(t *testing.T, got maker.CraftRequest) {
				if got.Mode != 2 || got.TargetItemId != 1082002 || got.UseCatalyst {
					t.Fatalf("unexpected create-with-upgrade request: %+v", got)
				}
			},
		},
		{
			name:  "MONSTER_CRYSTAL",
			bytes: makerSkillMonsterCrystalBytes,
			check: func(t *testing.T, got maker.CraftRequest) {
				if got.Mode != 3 || got.LeftoverItemId != 4000000 {
					t.Fatalf("unexpected monster-crystal request: %+v", got)
				}
			},
		},
		{
			name:  "DISASSEMBLE",
			bytes: makerSkillDisassembleBytes,
			check: func(t *testing.T, got maker.CraftRequest) {
				if got.Mode != 4 || got.EquipItemId != 1082002 || got.SlotPos != 5 {
					t.Fatalf("unexpected disassemble request: %+v", got)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var captured maker.CraftRequest
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&struct {
					Data struct {
						Attributes *maker.CraftRequest `json:"attributes"`
					} `json:"data"`
				}{Data: struct {
					Attributes *maker.CraftRequest `json:"attributes"`
				}{Attributes: &captured}}); err != nil {
					t.Errorf("decoding forwarded request: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data":{"type":"makerCrafts","id":"1","attributes":{"transactionId":"11111111-1111-1111-1111-111111111111"}}}`))
			}))
			defer srv.Close()
			t.Setenv("MAKER_SERVICE_URL", srv.URL+"/")

			s, ctx, cleanup := newCashItemUseTestSession(t, 778899)
			defer cleanup()

			rec := &makerRecorder{}
			MakerSkillHandleFunc(logrus.New(), ctx, rec.producer())(s, makerSkillReader(tc.bytes), map[string]interface{}{})

			if len(rec.announced) != 0 {
				t.Fatalf("expected no MAKER_RESULT write on acceptance, got %d", len(rec.announced))
			}
			tc.check(t, captured)
		})
	}
}

// craftErrorTestCases enumerates every PRD §5 stable code atlas-maker's POST
// /crafts can return (services/atlas-maker/atlas.com/maker/craft/errors.go).
var craftErrorTestCases = []struct {
	name   string
	status int
	code   string
}{
	{"recipe_not_found", http.StatusNotFound, maker.CodeRecipeNotFound},
	{"level_too_low", http.StatusUnprocessableEntity, maker.CodeLevelTooLow},
	{"skill_level_too_low", http.StatusUnprocessableEntity, maker.CodeSkillLevelTooLow},
	{"insufficient_materials", http.StatusUnprocessableEntity, maker.CodeInsufficientMaterials},
	{"missing_prerequisite_item", http.StatusUnprocessableEntity, maker.CodeMissingPrerequisiteItem},
	{"missing_prerequisite_quest", http.StatusUnprocessableEntity, maker.CodeMissingPrerequisiteQuest},
	{"insufficient_mesos", http.StatusUnprocessableEntity, maker.CodeInsufficientMesos},
	{"inventory_full", http.StatusUnprocessableEntity, maker.CodeInventoryFull},
	{"equip_not_found", http.StatusUnprocessableEntity, maker.CodeEquipNotFound},
	{"no_crystal_mapping", http.StatusUnprocessableEntity, maker.CodeNoCrystalMapping},
	{"craft_in_progress", http.StatusConflict, maker.CodeCraftInProgress},
	{"invalid_mode", http.StatusUnprocessableEntity, maker.CodeInvalidMode},
}

// TestMakerSkillHandlerWritesFailureOnRejection pins FR-5.2: no rejection
// code may leave the client UI locked -- every one of atlas-maker's PRD §5
// codes must still resolve to a written MAKER_RESULT failure.
func TestMakerSkillHandlerWritesFailureOnRejection(t *testing.T) {
	for _, tc := range craftErrorTestCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeCraftErrorResponse(w, tc.status, tc.code)
			}))
			defer srv.Close()
			t.Setenv("MAKER_SERVICE_URL", srv.URL+"/")

			s, ctx, cleanup := newCashItemUseTestSession(t, 778899)
			defer cleanup()

			rec := &makerRecorder{}
			MakerSkillHandleFunc(logrus.New(), ctx, rec.producer())(s, makerSkillReader(makerSkillCreateBytes), map[string]interface{}{})

			assertMakerResultFailedWritten(t, rec)
		})
	}
}

// TestMakerSkillHandlerWritesNothingOnAcceptance pins design §3.3: the
// success result is written from the saga's terminal event, never here.
func TestMakerSkillHandlerWritesNothingOnAcceptance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"type":"makerCrafts","id":"1","attributes":{"transactionId":"11111111-1111-1111-1111-111111111111"}}}`))
	}))
	defer srv.Close()
	t.Setenv("MAKER_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, 778899)
	defer cleanup()

	rec := &makerRecorder{}
	MakerSkillHandleFunc(logrus.New(), ctx, rec.producer())(s, makerSkillReader(makerSkillCreateBytes), map[string]interface{}{})

	if len(rec.announced) != 0 {
		t.Fatalf("expected no packet written on acceptance, got %d", len(rec.announced))
	}
}

// TestMakerSkillHandlerWritesFailureWhenMakerIsUnreachable pins that an
// upstream atlas-maker outage must not lock the UI either.
func TestMakerSkillHandlerWritesFailureWhenMakerIsUnreachable(t *testing.T) {
	// A closed server: any connection attempt fails at the transport level.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	srv.Close()
	t.Setenv("MAKER_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, 778899)
	defer cleanup()

	rec := &makerRecorder{}
	MakerSkillHandleFunc(logrus.New(), ctx, rec.producer())(s, makerSkillReader(makerSkillCreateBytes), map[string]interface{}{})

	assertMakerResultFailedWritten(t, rec)
}

func assertMakerResultFailedWritten(t *testing.T, rec *makerRecorder) {
	t.Helper()
	if len(rec.announced) != 1 {
		t.Fatalf("announced %d packets, want 1 (MAKER_RESULT failure)", len(rec.announced))
	}
	got := rec.announced[0]
	if got.writer != charcb.MakerResultWriter {
		t.Fatalf("announced writer = %q, want %q", got.writer, charcb.MakerResultWriter)
	}
	if len(got.body) != 4 {
		t.Fatalf("FAILED body length = %d, want 4 (nResult only)", len(got.body))
	}
}

// TestMakerSkillHandlerRequiresLogin pins that every applicable version's
// seed template gates MakerSkillHandle behind LoggedInValidator, like every
// other in-field op (this task's brief).
func TestMakerSkillHandlerRequiresLogin(t *testing.T) {
	templatesDir := filepath.Join("..", "..", "..", "..", "..", "..", "services", "atlas-configurations", "seed-data", "templates")
	versions := []string{
		"template_gms_72_1.json",
		"template_gms_79_1.json",
		"template_gms_83_1.json",
		"template_gms_84_1.json",
		"template_gms_87_1.json",
		"template_gms_92_1.json",
		"template_gms_95_1.json",
		"template_jms_185_1.json",
	}

	for _, v := range versions {
		t.Run(v, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(templatesDir, v))
			if err != nil {
				t.Fatalf("reading %s: %v", v, err)
			}
			var doc struct {
				Socket struct {
					Handlers []struct {
						Handler   string `json:"handler"`
						Validator string `json:"validator"`
					} `json:"handlers"`
				} `json:"socket"`
			}
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatalf("unmarshaling %s: %v", v, err)
			}
			for _, h := range doc.Socket.Handlers {
				if h.Handler != "MakerSkillHandle" {
					continue
				}
				if h.Validator != "LoggedInValidator" {
					t.Fatalf("%s: MakerSkillHandle validator = %q, want %q", v, h.Validator, "LoggedInValidator")
				}
				return
			}
			t.Fatalf("%s: no MakerSkillHandle entry found", v)
		})
	}
}
