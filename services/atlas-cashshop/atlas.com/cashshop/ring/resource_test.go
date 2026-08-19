// Package ring_test drives the read-only REST surface (ring.InitResource)
// through the REAL router over real HTTP requests -- task 24a's item 2, the
// HTTP-level cross-tenant isolation coverage GET /rings and GET
// /rings/{ringId} never had. It is an EXTERNAL test package, mirroring
// coupon/resource_test.go's shape and its
// TestAnotherTenantCanNeitherReadNorMutateTheseCoupons pattern (same harness:
// databasetest.NewInMemoryTenantDB, a real mux.Router, tenant identified only
// by request headers).
package ring_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-cashshop/ring"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

// newEnv migrates cash_rings and registers ring.InitResource on a real
// router, mirroring coupon_test.newEnv.
func newEnv(t *testing.T) (*httptest.Server, *gorm.DB, tenant.Model) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, ring.Migration)

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.ErrorLevel)

	r := mux.NewRouter()
	si := &testServerInformation{}
	ring.InitResource(si)(db)(r, l)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return srv, db, tm
}

func do(t *testing.T, srv *httptest.Server, tm tenant.Model, method, path string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, bytes.NewReader(nil))
	require.NoError(t, err)
	req.Header.Set("TENANT_ID", tm.Id().String())
	req.Header.Set("REGION", tm.Region())
	req.Header.Set("MAJOR_VERSION", fmt.Sprintf("%d", tm.MajorVersion()))
	req.Header.Set("MINOR_VERSION", fmt.Sprintf("%d", tm.MinorVersion()))

	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err)
	return resp, buf.Bytes()
}

func decodeDoc(t *testing.T, body []byte) jsonapi.Document {
	t.Helper()
	var doc jsonapi.Document
	require.NoError(t, json.Unmarshal(body, &doc))
	return doc
}

// seedPair inserts one pair directly via ring.CreatePair, bypassing the
// (write-side, atlas-cashshop-owned) purchase flow this read-only resource
// does not itself trigger.
func seedPair(t *testing.T, db *gorm.DB, tm tenant.Model, characterId, partnerCharacterId uint32) (uuid.UUID, uuid.UUID) {
	t.Helper()
	pairId, err := ring.CreatePair(db, tm.Id(), ring.TypeFriendship,
		ring.Half{CharacterId: characterId, AssetId: 1001, ItemTemplateId: 1112800},
		ring.Half{CharacterId: partnerCharacterId, AssetId: 1002, ItemTemplateId: 1112800},
	)
	require.NoError(t, err)

	rows, err := ring.GetByCharacterId(db, tm.Id(), characterId)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	return pairId, rows[0].Id()
}

// TestAnotherTenantCannotReadTheseRings is task 24a's item 2: the same
// cross-tenant isolation guarantee coupon/resource_test.go's
// TestAnotherTenantCanNeitherReadNorMutateTheseCoupons pins for the coupon
// surface, asserted at the REQUEST level -- the intruder's requests go
// through the same router, the same handlers, and the same
// byCharacterIdPagedProvider/GetById queries as the owner's, differing only
// in the tenant headers.
func TestAnotherTenantCannotReadTheseRings(t *testing.T) {
	srv, db, owner := newEnv(t)
	intruder, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)

	const ownerCharacterId = uint32(42)
	_, ringId := seedPair(t, db, owner, ownerCharacterId, 77)

	t.Run("the owner can list and read its own ring", func(t *testing.T) {
		resp, out := do(t, srv, owner, http.MethodGet, fmt.Sprintf("/rings?filter[characterId]=%d", ownerCharacterId))
		require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
		doc := decodeDoc(t, out)
		assert.EqualValues(t, 1, doc.Meta["total"])

		resp, out = do(t, srv, owner, http.MethodGet, "/rings/"+ringId.String())
		require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
	})

	t.Run("cannot list the ring", func(t *testing.T) {
		resp, out := do(t, srv, intruder, http.MethodGet, fmt.Sprintf("/rings?filter[characterId]=%d", ownerCharacterId))
		require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
		doc := decodeDoc(t, out)
		assert.EqualValues(t, 0, doc.Meta["total"], "another tenant's characterId filter must not surface the owner's ring")
	})

	t.Run("cannot read the ring by id", func(t *testing.T) {
		resp, _ := do(t, srv, intruder, http.MethodGet, "/rings/"+ringId.String())
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "another tenant must not be able to fetch the owner's ring by id")
	})

	t.Run("the owner's ring is untouched", func(t *testing.T) {
		resp, out := do(t, srv, owner, http.MethodGet, "/rings/"+ringId.String())
		require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
	})
}
