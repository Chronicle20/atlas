package pet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-pets/kafka/message"
	"atlas-pets/pet/exclude"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type paginateTestServerInformation struct{}

func (t *paginateTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *paginateTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &paginateTestServerInformation{}

func setupPetRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&paginateTestServerInformation{})(db)
	ri(r, l)
	return r
}

func requestPetsWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

// seedPet creates a pet owned by ownerId via the real processor.Create path
// (buffered, no Kafka emission) so TenantId/defaults are set exactly as
// production writes them.
func seedPet(t *testing.T, db *gorm.DB, ctx context.Context, ownerId uint32, templateId uint32) Model {
	t.Helper()
	b := NewModelBuilder(0, uint64(templateId)*100, templateId, "Pet", ownerId)
	i, err := b.Build()
	require.NoError(t, err)
	m, err := NewProcessor(testPaginateLogger(), ctx, db).Create(message.NewBuffer())(i)
	require.NoError(t, err)
	return m
}

func testPaginateLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

// TestGetPetsForCharacterPaginates drives GET /characters/{characterId}/pets
// through the real resource router, verifying the JSON:API paginated
// envelope, 400 on invalid paging params, empty-page handling past the last
// page, and that another character's pets are excluded from the total.
func TestGetPetsForCharacterPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, exclude.Migration)
	tenantId := uuid.New()
	ctx := databasetest.TenantContext(tenantId)

	seedPet(t, db, ctx, 1, 5000017)
	seedPet(t, db, ctx, 1, 5000018)
	seedPet(t, db, ctx, 1, 5000019)
	seedPet(t, db, ctx, 2, 5000020)

	srv := httptest.NewServer(setupPetRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/pets?page[number]=1&page[size]=2", srv.URL)
		req := requestPetsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"], "must exclude character 2's pet")
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/pets?page[size]=0", srv.URL)
		req := requestPetsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/pets?limit=5", srv.URL)
		req := requestPetsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/pets?page[number]=99&page[size]=2", srv.URL)
		req := requestPetsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 0)

		require.NotNil(t, doc.Links)
		require.Contains(t, doc.Links, "prev")
		assert.Contains(t, doc.Links["prev"].Href, "page%5Bnumber%5D=2")
		assert.NotContains(t, doc.Links, "next")
	})
}
