package area_info_test

import (
	"atlas-saga-orchestrator/area_info"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// putAreaInfoDoc is the JSON:API request document shape produced by
// requests.PutRequest for a RestModel.
type putAreaInfoDoc struct {
	Data struct {
		Attributes struct {
			Info string `json:"info"`
		} `json:"attributes"`
	} `json:"data"`
}

// TestPutIssuesPutWithInfo proves Processor.Put issues a PUT against the
// character-scoped area-info resource path, carrying the info string.
func TestPutIssuesPutWithInfo(t *testing.T) {
	const characterId = uint32(100100)
	const area = uint16(1)
	const info = "some persisted area info"

	var capturedPath string
	var capturedMethod string
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		capturedBody = b
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"data":{"id":"%d-%d","type":"area-infos","attributes":{"characterId":%d,"area":%d,"info":%q}}}`,
			characterId, area, characterId, area, info)
	}))
	defer srv.Close()
	t.Setenv("CHARACTER_URL_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	p := area_info.NewProcessor(l, ctx)
	if err := p.Put(characterId, area, info); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	if capturedMethod != http.MethodPut {
		t.Fatalf("expected method PUT, got %q", capturedMethod)
	}
	wantPath := fmt.Sprintf("/characters/%d/area-info/%d", characterId, area)
	if capturedPath != wantPath {
		t.Fatalf("expected path %q, got %q", wantPath, capturedPath)
	}

	var got putAreaInfoDoc
	if err := json.Unmarshal(capturedBody, &got); err != nil {
		t.Fatalf("failed to unmarshal request body: %v", err)
	}
	if got.Data.Attributes.Info != info {
		t.Fatalf("expected info=%q on the wire, got %q", info, got.Data.Attributes.Info)
	}
}

// TestPutPropagatesUpstreamFailure proves a non-2xx response from
// atlas-character surfaces as an error rather than being swallowed.
func TestPutPropagatesUpstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("CHARACTER_URL_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	p := area_info.NewProcessor(l, ctx)
	if err := p.Put(100100, 1, "info"); err == nil {
		t.Fatal("expected an error from a failing upstream response, got nil")
	}
}
