package macro_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-skills/kafka/message"
	"atlas-skills/macro"
	"atlas-skills/test"

	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupMacroRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := macro.InitResource(&testServerInformation{})(db)
	ri(r, l)
	return r
}

func requestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// TestGetSkillMacrosPaginates drives GET /characters/{characterId}/macros
// through the real resource router, verifying the JSON:API paginated
// envelope, 400 on invalid paging params, and empty-page handling past the
// last page.
func TestGetSkillMacrosPaginates(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	l, _ := logtest.NewNullLogger()
	processor := macro.NewProcessor(l, ctx, db)

	transactionId := uuid.New()
	worldId := world.Id(0)
	characterId := uint32(42)
	mb := message.NewBuffer()

	var seeded []macro.Model
	for i := 0; i < 3; i++ {
		m, err := macro.NewModelBuilder().
			SetId(uint32(i)).
			SetName(fmt.Sprintf("Macro %d", i)).
			SetSkillId1(skillconst.Id(1001001 + i)).
			Build()
		require.NoError(t, err)
		seeded = append(seeded, m)
	}
	_, err := processor.Update(mb)(transactionId, worldId, characterId, seeded)
	require.NoError(t, err)

	srv := httptest.NewServer(setupMacroRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/42/macros?page[number]=1&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/42/macros?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/42/macros?limit=5", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/42/macros?page[number]=99&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

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
