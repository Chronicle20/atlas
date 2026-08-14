package pending_change

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type resourceTestServerInfo struct{}

func (t *resourceTestServerInfo) GetBaseURL() string { return "http://localhost:8080" }
func (t *resourceTestServerInfo) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &resourceTestServerInfo{}

func setupPendingChangeResourceRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&resourceTestServerInfo{})(db)
	ri(r, l)
	return r
}

func pendingChangeRequestWithTenant(t *testing.T, method, url string, body []byte) *http.Request {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader([]byte{})
	}
	req, err := http.NewRequest(method, url, reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", testTenantModel.Id().String())
	req.Header.Set("REGION", testTenantModel.Region())
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func createBody(t *testing.T, in CreateInputRestModel) []byte {
	t.Helper()
	b, err := jsonapi.Marshal(in)
	require.NoError(t, err)
	return b
}

// seedChar creates a plain character with no pending change.
func seedChar(t *testing.T, db *gorm.DB) uint32 {
	t.Helper()
	return seedCharacter(t, db, "Golf", world.Id(0))
}

// seedCharWithPending creates a character already holding a PENDING name
// change, so a second NAME_CHANGE request collides with
// idx_pc_one_pending_per_type.
func seedCharWithPending(t *testing.T, db *gorm.DB) uint32 {
	t.Helper()
	characterId := seedCharacter(t, db, "Hotel", world.Id(0))
	_, err := NewProcessor(testLogger(t), testContext(t), db).
		CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Existing", world.Id(0), nil)
	require.NoError(t, err)
	return characterId
}

// TestCreatePendingChangeMapsRejectionsToStatusCodes drives POST
// /characters/{id}/pending-changes through the real resource router,
// asserting the processor's typed rejections land on the documented HTTP
// status (and, for the reason-carrying ones, the reason lands in the error
// body's detail).
func TestCreatePendingChangeMapsRejectionsToStatusCodes(t *testing.T) {
	cases := []struct {
		name   string
		seed   func(t *testing.T, db *gorm.DB) uint32
		input  CreateInputRestModel
		want   int
		reason string
	}{
		{
			name:  "unknown character",
			input: CreateInputRestModel{Type: TypeNameChange, RequestedName: "India"},
			want:  http.StatusNotFound,
		},
		{
			name:   "invalid name",
			seed:   seedChar,
			input:  CreateInputRestModel{Type: TypeNameChange, RequestedName: "ab"},
			want:   http.StatusUnprocessableEntity,
			reason: "name_invalid_length",
		},
		{
			name:   "already pending",
			seed:   seedCharWithPending,
			input:  CreateInputRestModel{Type: TypeNameChange, RequestedName: "Juliet"},
			want:   http.StatusConflict,
			reason: "already_pending",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newProcessorTestDB(t)
			var characterId uint32 = 999999
			if tc.seed != nil {
				characterId = tc.seed(t, db)
			}

			router := setupPendingChangeResourceRouter(db)
			srv := httptest.NewServer(router)
			defer srv.Close()

			url := fmt.Sprintf("%s/characters/%d/pending-changes", srv.URL, characterId)
			req := pendingChangeRequestWithTenant(t, http.MethodPost, url, createBody(t, tc.input))

			resp, err := (&http.Client{}).Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			require.Equal(t, tc.want, resp.StatusCode)

			if tc.reason != "" {
				var doc struct {
					Errors []struct {
						Detail string `json:"detail"`
					} `json:"errors"`
				}
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
				require.Len(t, doc.Errors, 1)
				require.Equal(t, tc.reason, doc.Errors[0].Detail)
			}
		})
	}
}

// TestDeletePendingChangeIsConflictOnceTerminal drives DELETE
// /characters/{id}/pending-changes/{id} through the real resource router:
// the first cancel moves the record to CANCELLED (204), and a redelivered or
// repeated cancel on the now-terminal record is a 409, never a silent
// success or a second refund.
func TestDeletePendingChangeIsConflictOnceTerminal(t *testing.T) {
	db := newProcessorTestDB(t)
	characterId := seedCharacter(t, db, "Kilo", world.Id(0))

	m, err := NewProcessor(testLogger(t), testContext(t), db).
		CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Lima", world.Id(0), nil)
	require.NoError(t, err)

	router := setupPendingChangeResourceRouter(db)
	srv := httptest.NewServer(router)
	defer srv.Close()

	url := fmt.Sprintf("%s/characters/%d/pending-changes/%s", srv.URL, characterId, m.Id())

	first := pendingChangeRequestWithTenant(t, http.MethodDelete, url, nil)
	resp, err := (&http.Client{}).Do(first)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	got, err := NewProcessor(testLogger(t), testContext(t), db).GetById(m.Id())
	require.NoError(t, err)
	require.Equal(t, StatusCancelled, got.Status())

	second := pendingChangeRequestWithTenant(t, http.MethodDelete, url, nil)
	resp2, err := (&http.Client{}).Do(second)
	require.NoError(t, err)
	_ = resp2.Body.Close()
	require.Equal(t, http.StatusConflict, resp2.StatusCode)
}
