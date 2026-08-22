package configuration_test

import (
	"atlas-tenants/configuration"
	tenants "atlas-tenants/tenant"
	"atlas-tenants/test"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// playerNpcConfigHandlerTestDB builds the same in-memory sqlite database
// rankingsHandlerTestDB uses (see rankings_handler_test.go), plus the
// outbox_entries table the …AndEmit methods need.
func playerNpcConfigHandlerTestDB(t *testing.T) (*httptest.Server, *gorm.DB) {
	t.Helper()
	db := test.SetupTestDB(t)
	if err := outbox.Migration(db); err != nil {
		t.Fatalf("outbox migration: %v", err)
	}
	logger, _ := logtest.NewNullLogger()

	router := mux.NewRouter()
	configuration.RegisterRoutes(db)(testServerInformation{})(router, logger)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	t.Cleanup(func() { test.CleanupTestDB(db) })
	return srv, db
}

func doPlayerNpcConfigRequest(t *testing.T, method, url string, body []byte) *http.Response {
	t.Helper()
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

type playerNpcConfigAttributes struct {
	InitialX          int16 `json:"initialX"`
	InitialY          int16 `json:"initialY"`
	AreaX             int16 `json:"areaX"`
	AreaY             int16 `json:"areaY"`
	AreaSteps         int   `json:"areaSteps"`
	OrganizeArea      bool  `json:"organizeArea"`
	AutoDeployEnabled bool  `json:"autoDeployEnabled"`
}

type playerNpcConfigDoc struct {
	Data struct {
		Type       string                    `json:"type"`
		Id         string                    `json:"id"`
		Attributes playerNpcConfigAttributes `json:"attributes"`
	} `json:"data"`
}

// TestPlayerNpcConfigHandlerWireRoundTrip drives the player-npcs
// configuration resource through its actual HTTP handlers
// (CreatePlayerNpcConfigHandler, GetPlayerNpcConfigHandler,
// UpdatePlayerNpcConfigHandler, DeletePlayerNpcConfigHandler) and the real
// JSON:API codec (jsonapi.Unmarshal on decode via server.ParseInput,
// jsonapi.MarshalToStruct on encode via server.MarshalResponse) — not the
// processor or TransformPlayerNpcConfig / ExtractPlayerNpcConfig called
// directly, mirroring TestRankingsHandlerWireRoundTrip.
func TestPlayerNpcConfigHandlerWireRoundTrip(t *testing.T) {
	srv, db := playerNpcConfigHandlerTestDB(t)
	tenantId := uuid.New()
	if err := db.Create(&tenants.Entity{
		ID:           tenantId,
		Name:         "player-npc-config-round-trip",
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
	}).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	playerNpcConfigURL := fmt.Sprintf("%s/tenants/%s/configurations/player-npcs", srv.URL, tenantId)

	// unconfigured: GET before any create returns 404.
	notFoundResp := doPlayerNpcConfigRequest(t, http.MethodGet, playerNpcConfigURL, nil)
	defer func() { _ = notFoundResp.Body.Close() }()
	if notFoundResp.StatusCode != http.StatusNotFound {
		t.Fatalf("get before create status = %d, want 404", notFoundResp.StatusCode)
	}

	// create then get: POST the seven values, GET returns them unchanged.
	envelope := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "player-npcs",
			"attributes": map[string]interface{}{
				"initialX":          100,
				"initialY":          200,
				"areaX":             320,
				"areaY":             160,
				"areaSteps":         4,
				"organizeArea":      true,
				"autoDeployEnabled": true,
			},
		},
	}
	postBody, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal post body: %v", err)
	}

	postResp := doPlayerNpcConfigRequest(t, http.MethodPost, playerNpcConfigURL, postBody)
	defer func() { _ = postResp.Body.Close() }()
	if postResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", postResp.StatusCode)
	}

	postRaw := new(bytes.Buffer)
	if _, err := postRaw.ReadFrom(postResp.Body); err != nil {
		t.Fatalf("read create body: %v", err)
	}
	var postDoc playerNpcConfigDoc
	if err := json.Unmarshal(postRaw.Bytes(), &postDoc); err != nil {
		t.Fatalf("decode create body: %v, body=%s", err, postRaw.String())
	}

	if postDoc.Data.Type != "player-npcs" {
		t.Fatalf("create data.type = %q, want %q", postDoc.Data.Type, "player-npcs")
	}
	if postDoc.Data.Id == "" {
		t.Fatal("create did not assign an id")
	}
	want := playerNpcConfigAttributes{
		InitialX:          100,
		InitialY:          200,
		AreaX:             320,
		AreaY:             160,
		AreaSteps:         4,
		OrganizeArea:      true,
		AutoDeployEnabled: true,
	}
	if postDoc.Data.Attributes != want {
		t.Fatalf("create data.attributes = %+v, want %+v", postDoc.Data.Attributes, want)
	}
	// type fidelity: organizeArea/autoDeployEnabled survive as booleans, not
	// strings.
	if !strings.Contains(postRaw.String(), `"organizeArea":true`) {
		t.Fatalf("create response organizeArea is not a literal boolean; body=%s", postRaw.String())
	}
	if !strings.Contains(postRaw.String(), `"autoDeployEnabled":true`) {
		t.Fatalf("create response autoDeployEnabled is not a literal boolean; body=%s", postRaw.String())
	}

	getResp := doPlayerNpcConfigRequest(t, http.MethodGet, playerNpcConfigURL, nil)
	defer func() { _ = getResp.Body.Close() }()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getResp.StatusCode)
	}
	getRaw := new(bytes.Buffer)
	if _, err := getRaw.ReadFrom(getResp.Body); err != nil {
		t.Fatalf("read get body: %v", err)
	}
	var getDoc playerNpcConfigDoc
	if err := json.Unmarshal(getRaw.Bytes(), &getDoc); err != nil {
		t.Fatalf("decode get body: %v, body=%s", err, getRaw.String())
	}
	if getDoc.Data.Type != "player-npcs" {
		t.Fatalf("get data.type = %q, want %q", getDoc.Data.Type, "player-npcs")
	}
	if getDoc.Data.Attributes != want {
		t.Fatalf("get data.attributes = %+v, want %+v", getDoc.Data.Attributes, want)
	}

	// update: PATCH changes areaSteps only; the other six are unchanged.
	updateEnvelope := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "player-npcs",
			"id":   getDoc.Data.Id,
			"attributes": map[string]interface{}{
				"initialX":          100,
				"initialY":          200,
				"areaX":             320,
				"areaY":             160,
				"areaSteps":         7,
				"organizeArea":      true,
				"autoDeployEnabled": true,
			},
		},
	}
	patchBody, err := json.Marshal(updateEnvelope)
	if err != nil {
		t.Fatalf("marshal patch body: %v", err)
	}
	patchResp := doPlayerNpcConfigRequest(t, http.MethodPatch, playerNpcConfigURL, patchBody)
	defer func() { _ = patchResp.Body.Close() }()
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d, want 200", patchResp.StatusCode)
	}
	patchRaw := new(bytes.Buffer)
	if _, err := patchRaw.ReadFrom(patchResp.Body); err != nil {
		t.Fatalf("read update body: %v", err)
	}
	var patchDoc playerNpcConfigDoc
	if err := json.Unmarshal(patchRaw.Bytes(), &patchDoc); err != nil {
		t.Fatalf("decode update body: %v, body=%s", err, patchRaw.String())
	}
	wantUpdated := want
	wantUpdated.AreaSteps = 7
	if patchDoc.Data.Attributes != wantUpdated {
		t.Fatalf("update data.attributes = %+v, want %+v", patchDoc.Data.Attributes, wantUpdated)
	}

	getAfterUpdateResp := doPlayerNpcConfigRequest(t, http.MethodGet, playerNpcConfigURL, nil)
	defer func() { _ = getAfterUpdateResp.Body.Close() }()
	getAfterUpdateRaw := new(bytes.Buffer)
	if _, err := getAfterUpdateRaw.ReadFrom(getAfterUpdateResp.Body); err != nil {
		t.Fatalf("read get-after-update body: %v", err)
	}
	var getAfterUpdateDoc playerNpcConfigDoc
	if err := json.Unmarshal(getAfterUpdateRaw.Bytes(), &getAfterUpdateDoc); err != nil {
		t.Fatalf("decode get-after-update body: %v, body=%s", err, getAfterUpdateRaw.String())
	}
	if getAfterUpdateDoc.Data.Attributes != wantUpdated {
		t.Fatalf("get-after-update data.attributes = %+v, want %+v", getAfterUpdateDoc.Data.Attributes, wantUpdated)
	}

	// delete then get: DELETE then GET returns 404.
	deleteResp := doPlayerNpcConfigRequest(t, http.MethodDelete, playerNpcConfigURL, nil)
	defer func() { _ = deleteResp.Body.Close() }()
	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", deleteResp.StatusCode)
	}

	getAfterDeleteResp := doPlayerNpcConfigRequest(t, http.MethodGet, playerNpcConfigURL, nil)
	defer func() { _ = getAfterDeleteResp.Body.Close() }()
	if getAfterDeleteResp.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", getAfterDeleteResp.StatusCode)
	}
}
