package factory_test

import (
	"atlas-channel/character/factory"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testContext(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return tenant.WithContext(context.Background(), ten)
}

// seedArgs pins the session-supplied arguments SeedCharacter is called with
// across the tests below.
var (
	seedAccountId    = uint32(1001)
	seedWorldId      = world.Id(0)
	seedName         = "TestChar"
	seedJobIndex     = uint32(0)
	seedSubJobIndex  = uint16(0)
	seedFace         = uint32(20000)
	seedHair         = uint32(30000)
	seedColor        = uint32(0)
	seedSkinColor    = uint32(0)
	seedGender       = byte(0)
	seedTop          = uint32(1040002)
	seedBottom       = uint32(1060002)
	seedShoes        = uint32(1072001)
	seedWeapon       = uint32(1302000)
	seedStrength     = byte(12)
	seedDexterity    = byte(5)
	seedIntelligence = byte(4)
	seedLuck         = byte(4)
)

func callSeedCharacter(ctx context.Context) (string, error) {
	log, _ := test.NewNullLogger()
	p := factory.NewProcessor(log, ctx)
	return p.SeedCharacter(seedAccountId, seedWorldId, seedName, seedJobIndex, seedSubJobIndex,
		seedFace, seedHair, seedColor, seedSkinColor, seedGender,
		seedTop, seedBottom, seedShoes, seedWeapon,
		seedStrength, seedDexterity, seedIntelligence, seedLuck)
}

func TestSeedCharacterSendsSessionSuppliedIds(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		captured = string(buf)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"type":"characters","id":"tx-1","attributes":{"transactionId":"tx-1"}}}`))
	}))
	defer srv.Close()
	t.Setenv("CHARACTER_FACTORY_SERVICE_URL", srv.URL+"/")

	ctx := testContext(t)
	if _, err := callSeedCharacter(ctx); err != nil {
		t.Fatal(err)
	}

	wantAccountId := `"accountId":` + strconv.Itoa(int(seedAccountId))
	if !strings.Contains(captured, wantAccountId) {
		t.Errorf("expected request body to contain %q, got %q", wantAccountId, captured)
	}
	wantWorldId := `"worldId":` + strconv.Itoa(int(seedWorldId))
	if !strings.Contains(captured, wantWorldId) {
		t.Errorf("expected request body to contain %q, got %q", wantWorldId, captured)
	}
	if !strings.Contains(captured, `"level":1`) {
		t.Errorf("expected request body to contain default level 1, got %q", captured)
	}
	if !strings.Contains(captured, `"hp":50`) {
		t.Errorf("expected request body to contain default hp 50, got %q", captured)
	}
	if !strings.Contains(captured, `"mp":5`) {
		t.Errorf("expected request body to contain default mp 5, got %q", captured)
	}
	if !strings.Contains(captured, `"mapId":0`) {
		t.Errorf("expected request body to contain default mapId 0, got %q", captured)
	}
}

func TestSeedCharacterCarriesTenantHeader(t *testing.T) {
	var capturedHeader http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Clone()
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"type":"characters","id":"tx-1","attributes":{"transactionId":"tx-1"}}}`))
	}))
	defer srv.Close()
	t.Setenv("CHARACTER_FACTORY_SERVICE_URL", srv.URL+"/")

	ctx := testContext(t)
	ten, _ := tenant.FromContext(ctx)()
	if _, err := callSeedCharacter(ctx); err != nil {
		t.Fatal(err)
	}

	if capturedHeader.Get(tenant.ID) != ten.Id().String() {
		t.Errorf("expected tenant header [%s] to be [%s], got [%s]", tenant.ID, ten.Id().String(), capturedHeader.Get(tenant.ID))
	}
}

func TestSeedCharacterPostsToCharactersSeed(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"type":"characters","id":"tx-1","attributes":{"transactionId":"tx-1"}}}`))
	}))
	defer srv.Close()
	t.Setenv("CHARACTER_FACTORY_SERVICE_URL", srv.URL+"/")

	ctx := testContext(t)
	if _, err := callSeedCharacter(ctx); err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(capturedPath, "characters/seed") {
		t.Errorf("expected request path to end with [characters/seed], got [%s]", capturedPath)
	}
}

func TestSeedCharacterReturnsTransactionId(t *testing.T) {
	t.Run("accepted", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"type":"characters","id":"tx-42","attributes":{"transactionId":"tx-42"}}}`))
		}))
		defer srv.Close()
		t.Setenv("CHARACTER_FACTORY_SERVICE_URL", srv.URL+"/")

		ctx := testContext(t)
		txId, err := callSeedCharacter(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if txId != "tx-42" {
			t.Errorf("expected transactionId [tx-42], got [%s]", txId)
		}
	})

	t.Run("rejected", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errors":[{"status":"400","title":"invalid character"}]}`))
		}))
		defer srv.Close()
		t.Setenv("CHARACTER_FACTORY_SERVICE_URL", srv.URL+"/")

		ctx := testContext(t)
		txId, err := callSeedCharacter(ctx)
		if err == nil {
			t.Fatal("expected an error for a rejected seed request, got nil")
		}
		if !errors.Is(err, requests.ErrBadRequest) {
			t.Errorf("expected err to be requests.ErrBadRequest so the caller can classify the rejection, got %v", err)
		}
		if txId != "" {
			t.Errorf("expected empty transactionId on error, got [%s]", txId)
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		srv.Close()
		t.Setenv("CHARACTER_FACTORY_SERVICE_URL", srv.URL+"/")

		ctx := testContext(t)
		txId, err := callSeedCharacter(ctx)
		if err == nil {
			t.Fatal("expected an error when the factory is unreachable, got nil")
		}
		if txId != "" {
			t.Errorf("expected empty transactionId on error, got [%s]", txId)
		}
	})
}

// mapleLifeArgs pins the player-chosen arguments CreateMapleLife is called
// with across the tests below. Each numeric value is distinct and non-zero
// so a positional argument mix-up fails a test.
var (
	mapleLifeAccountId    = uint32(2001)
	mapleLifeWorldId      = world.Id(2)
	mapleLifeName         = "MapleChar"
	mapleLifeClassOrdinal = uint32(3)
	mapleLifeGender       = byte(1)
	mapleLifeFace         = uint32(21000)
	mapleLifeHair         = uint32(31000)
	mapleLifeHairColor    = uint32(4)
	mapleLifeSkinColor    = byte(5)
	mapleLifeSP           = byte(6)
)

func callCreateMapleLife(ctx context.Context) (string, error) {
	log, _ := test.NewNullLogger()
	p := factory.NewProcessor(log, ctx)
	return p.CreateMapleLife(mapleLifeAccountId, mapleLifeWorldId, mapleLifeName, mapleLifeClassOrdinal, mapleLifeGender,
		mapleLifeFace, mapleLifeHair, mapleLifeHairColor, mapleLifeSkinColor, mapleLifeSP)
}

func TestCreateMapleLifePostsTheChosenValues(t *testing.T) {
	var capturedPath string
	var capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if r.Method != http.MethodPost {
			t.Errorf("expected method [%s], got [%s]", http.MethodPost, r.Method)
		}
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		capturedBody = string(buf)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"type":"characters","id":"tx-ml-1","attributes":{"transactionId":"tx-ml-1"}}}`))
	}))
	defer srv.Close()
	t.Setenv("CHARACTER_FACTORY_SERVICE_URL", srv.URL+"/")

	ctx := testContext(t)
	txId, err := callCreateMapleLife(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(capturedPath, "factory/characters/maple-life") {
		t.Errorf("expected request path to end with [factory/characters/maple-life], got [%s]", capturedPath)
	}

	wantFields := map[string]string{
		"accountId":    strconv.Itoa(int(mapleLifeAccountId)),
		"worldId":      strconv.Itoa(int(mapleLifeWorldId)),
		"classOrdinal": strconv.Itoa(int(mapleLifeClassOrdinal)),
		"gender":       strconv.Itoa(int(mapleLifeGender)),
		"face":         strconv.Itoa(int(mapleLifeFace)),
		"hair":         strconv.Itoa(int(mapleLifeHair)),
		"hairColor":    strconv.Itoa(int(mapleLifeHairColor)),
		"skinColor":    strconv.Itoa(int(mapleLifeSkinColor)),
		"sp":           strconv.Itoa(int(mapleLifeSP)),
	}
	for k, v := range wantFields {
		want := `"` + k + `":` + v
		if !strings.Contains(capturedBody, want) {
			t.Errorf("expected request body to contain %q, got %q", want, capturedBody)
		}
	}
	wantName := `"name":"` + mapleLifeName + `"`
	if !strings.Contains(capturedBody, wantName) {
		t.Errorf("expected request body to contain %q, got %q", wantName, capturedBody)
	}

	if txId != "tx-ml-1" {
		t.Errorf("expected transactionId [tx-ml-1], got [%s]", txId)
	}
}

func TestCreateMapleLifeSurfacesStatuses(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		check      func(t *testing.T, err error)
	}{
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			check: func(t *testing.T, err error) {
				if !errors.Is(err, requests.ErrBadRequest) {
					t.Errorf("expected err to be requests.ErrBadRequest, got %v", err)
				}
			},
		},
		{
			name:       "conflict",
			statusCode: http.StatusConflict,
			check: func(t *testing.T, err error) {
				if !errors.Is(err, requests.ErrConflict) {
					t.Errorf("expected err to be requests.ErrConflict, got %v", err)
				}
			},
		},
		{
			name:       "bad gateway",
			statusCode: http.StatusBadGateway,
			check: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected a non-nil error for a 502 response")
				}
			},
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			check: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected a non-nil error for a 500 response")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer srv.Close()
			t.Setenv("CHARACTER_FACTORY_SERVICE_URL", srv.URL+"/")

			ctx := testContext(t)
			txId, err := callCreateMapleLife(ctx)
			tc.check(t, err)
			if txId != "" {
				t.Errorf("expected empty transactionId on error, got [%s]", txId)
			}
		})
	}
}
