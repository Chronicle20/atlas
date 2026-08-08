package templates

import (
	"atlas-configurations/templates/socket"
	"atlas-configurations/templates/socket/handler"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// testEntity is a SQLite-compatible version of Entity for testing
type testEntity struct {
	Id           uuid.UUID       `gorm:"type:text;primaryKey"`
	Region       string          `gorm:"not null"`
	MajorVersion uint16          `gorm:"not null"`
	MinorVersion uint16          `gorm:"not null"`
	Data         json.RawMessage `gorm:"type:text;not null"`
}

func (testEntity) TableName() string {
	return "templates"
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Use SQLite-compatible schema
	err = db.AutoMigrate(&testEntity{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

func createTestRestModel(region string, majorVersion, minorVersion uint16) RestModel {
	return RestModel{
		Region:       region,
		MajorVersion: majorVersion,
		MinorVersion: minorVersion,
		UsesPin:      true,
	}
}

func TestProcessor_GetAll_Empty(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	paged, err := p.AllProvider(model.Page{Number: 1, Size: 250})()
	results := paged.Items
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestProcessor_GetAll_WithData(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	// Create test data
	input1 := createTestRestModel("GMS", 83, 1)
	input2 := createTestRestModel("SEA", 83, 2)

	_, err := p.Create(input1)
	if err != nil {
		t.Fatalf("failed to create first template: %v", err)
	}

	_, err = p.Create(input2)
	if err != nil {
		t.Fatalf("failed to create second template: %v", err)
	}

	paged, err := p.AllProvider(model.Page{Number: 1, Size: 250})()
	results := paged.Items
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestProcessor_Create(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	input := createTestRestModel("GMS", 83, 1)

	id, err := p.Create(input)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	if id == uuid.Nil {
		t.Error("expected non-nil UUID")
	}

	// Verify it was created
	result, err := p.GetById(id)
	if err != nil {
		t.Fatalf("failed to get created template: %v", err)
	}

	if result.Region != input.Region {
		t.Errorf("expected region '%s', got '%s'", input.Region, result.Region)
	}
	if result.MajorVersion != input.MajorVersion {
		t.Errorf("expected majorVersion %d, got %d", input.MajorVersion, result.MajorVersion)
	}
	if result.MinorVersion != input.MinorVersion {
		t.Errorf("expected minorVersion %d, got %d", input.MinorVersion, result.MinorVersion)
	}
	if result.UsesPin != input.UsesPin {
		t.Errorf("expected usesPin %v, got %v", input.UsesPin, result.UsesPin)
	}
}

func TestProcessor_GetById(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	input := createTestRestModel("GMS", 83, 1)
	id, err := p.Create(input)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	result, err := p.GetById(id)
	if err != nil {
		t.Fatalf("failed to get template: %v", err)
	}

	if result.Id != id.String() {
		t.Errorf("expected id '%s', got '%s'", id.String(), result.Id)
	}
	if result.Region != input.Region {
		t.Errorf("expected region '%s', got '%s'", input.Region, result.Region)
	}
}

func TestProcessor_GetById_NotFound(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	nonExistentId := uuid.New()
	_, err := p.GetById(nonExistentId)
	if err == nil {
		t.Error("expected error for non-existent template")
	}
}

func TestProcessor_GetByRegionAndVersion(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	input := createTestRestModel("GMS", 83, 1)
	_, err := p.Create(input)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	result, err := p.GetByRegionAndVersion("GMS", 83, 1)
	if err != nil {
		t.Fatalf("failed to get template by region/version: %v", err)
	}

	if result.Region != "GMS" {
		t.Errorf("expected region 'GMS', got '%s'", result.Region)
	}
	if result.MajorVersion != 83 {
		t.Errorf("expected majorVersion 83, got %d", result.MajorVersion)
	}
	if result.MinorVersion != 1 {
		t.Errorf("expected minorVersion 1, got %d", result.MinorVersion)
	}
}

func TestProcessor_GetByRegionAndVersion_NotFound(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	_, err := p.GetByRegionAndVersion("NONEXISTENT", 99, 99)
	if err == nil {
		t.Error("expected error for non-existent region/version")
	}
}

func TestProcessor_UpdateById(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	// Create initial template
	input := createTestRestModel("GMS", 83, 1)
	id, err := p.Create(input)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Update the template
	updated := createTestRestModel("SEA", 84, 2)
	updated.UsesPin = false
	err = p.UpdateById(id, updated)
	if err != nil {
		t.Fatalf("failed to update template: %v", err)
	}

	// Verify the update
	result, err := p.GetById(id)
	if err != nil {
		t.Fatalf("failed to get updated template: %v", err)
	}

	if result.Region != updated.Region {
		t.Errorf("expected region '%s', got '%s'", updated.Region, result.Region)
	}
	if result.MajorVersion != updated.MajorVersion {
		t.Errorf("expected majorVersion %d, got %d", updated.MajorVersion, result.MajorVersion)
	}
	if result.UsesPin != updated.UsesPin {
		t.Errorf("expected usesPin %v, got %v", updated.UsesPin, result.UsesPin)
	}
}

func TestProcessor_UpdateById_NotFound(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	nonExistentId := uuid.New()
	input := createTestRestModel("GMS", 83, 1)
	err := p.UpdateById(nonExistentId, input)
	if err == nil {
		t.Error("expected error for non-existent template")
	}
}

func TestProcessor_DeleteById(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	// Create a template
	input := createTestRestModel("GMS", 83, 1)
	id, err := p.Create(input)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Verify it exists
	_, err = p.GetById(id)
	if err != nil {
		t.Fatalf("template should exist before delete: %v", err)
	}

	// Delete it
	err = p.DeleteById(id)
	if err != nil {
		t.Fatalf("failed to delete template: %v", err)
	}

	// Verify it's gone
	_, err = p.GetById(id)
	if err == nil {
		t.Error("expected error for deleted template")
	}
}

func TestProcessor_DeleteById_NotFound(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	nonExistentId := uuid.New()
	err := p.DeleteById(nonExistentId)
	if err == nil {
		t.Error("expected error for non-existent template")
	}
}

func TestMake(t *testing.T) {
	testId := uuid.New()
	testData := RestModel{
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
		UsesPin:      true,
	}
	jsonData, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("failed to marshal test data: %v", err)
	}

	entity := Entity{
		Id:           testId,
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
		Data:         jsonData,
	}

	result, err := Make(entity)
	if err != nil {
		t.Fatalf("Make failed: %v", err)
	}

	if result.Id != testId.String() {
		t.Errorf("expected id '%s', got '%s'", testId.String(), result.Id)
	}
	if result.Region != testData.Region {
		t.Errorf("expected region '%s', got '%s'", testData.Region, result.Region)
	}
	if result.MajorVersion != testData.MajorVersion {
		t.Errorf("expected majorVersion %d, got %d", testData.MajorVersion, result.MajorVersion)
	}
	if result.UsesPin != testData.UsesPin {
		t.Errorf("expected usesPin %v, got %v", testData.UsesPin, result.UsesPin)
	}
}

func TestMake_InvalidJSON(t *testing.T) {
	entity := Entity{
		Id:           uuid.New(),
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
		Data:         json.RawMessage(`{invalid json`),
	}

	_, err := Make(entity)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// Design §5 step 2: Entity.Data must equal canonicalBytes(input) exactly.
// This is the pin that keeps re-seed producing a row byte-identical to what a
// fresh boot seed of the same file would produce. If Create ever grows a
// transformation canonicalBytes lacks, re-seeded rows report drift the instant
// they are reset, and this test is what catches it.
func TestCanonicalBytesMatchesCreate(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	in := createTestRestModel("GMS", 83, 1)

	id, err := p.Create(in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	want, err := canonicalBytes(in)
	if err != nil {
		t.Fatalf("canonicalBytes: %v", err)
	}

	var e Entity
	if err := db.Where("id = ?", id).First(&e).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(e.Data) != string(want) {
		t.Errorf("Entity.Data != canonicalBytes\n got: %s\nwant: %s", e.Data, want)
	}
}

// canonicalBytes must reject an invalid socket configuration with the same
// error type Create surfaces, so the re-seed handler's 400 branch can reuse
// the existing validationFailureError rendering verbatim.
func TestCanonicalBytesSurfacesValidationFailure(t *testing.T) {
	in := createTestRestModel("GMS", 83, 1)
	// A handler with no validator is silently dropped by the channel
	// dispatcher, so socket.Validate rejects it.
	in.Socket = socket.RestModel{
		Handlers: []handler.RestModel{{OpCode: "0x01", Handler: "SomeHandle"}},
	}

	_, err := canonicalBytes(in)
	if err == nil {
		t.Fatalf("canonicalBytes accepted an invalid socket configuration")
	}
	var ve *validationFailureError
	if !errors.As(err, &ve) {
		t.Fatalf("error type = %T, want *validationFailureError", err)
	}
}

// THE load-bearing test (PRD acceptance criteria, design §5). Every shipped
// seed file, created into a fresh database and read back through the view
// provider, must report NO drift. If Create or Normalize perturbs anything the
// revision sees, every template reports permanent phantom drift and the badge
// becomes noise. Table-driven over the directory so a new version bring-up is
// covered without editing this test.
func TestShippedSeedsReportNoDrift(t *testing.T) {
	l := testLogger()
	catalog := LoadCatalog(l, seedTemplatesDir())
	if catalog.Len() == 0 {
		t.Fatalf("no seed templates found at %s - a wrong path would make this test vacuously pass", seedTemplatesDir())
	}

	for _, entry := range catalog.Entries() {
		t.Run(entry.FileName, func(t *testing.T) {
			db := setupTestDB(t)
			ctx := context.Background()
			p := NewProcessor(l, ctx, db).WithCatalog(catalog)

			id, err := p.Create(entry.Model)
			if err != nil {
				t.Fatalf("Create(%s): %v", entry.FileName, err)
			}

			v, err := p.ViewByIdProvider(id)()
			if err != nil {
				t.Fatalf("ViewByIdProvider: %v", err)
			}

			if v.ShippedRevision != entry.Revision {
				t.Errorf("ShippedRevision = %q, want %q", v.ShippedRevision, entry.Revision)
			}
			if v.StoredRevision != entry.Revision {
				t.Errorf("StoredRevision = %q, want %q (Marshal∘Unmarshal is not the identity on this document - see design §5)", v.StoredRevision, entry.Revision)
			}
			if v.SeedDrift {
				t.Errorf("SeedDrift = true for a freshly seeded template")
			}
		})
	}
}

// FR-2.3: a stored row whose content no longer matches the shipped file drifts.
func TestDriftDetectedAfterMutation(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	catalog := LoadCatalog(l, seedTemplatesDir())

	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}

	p := NewProcessor(l, ctx, db).WithCatalog(catalog)
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Mutate the stored document out from under the row, the way a UI PATCH
	// or an out-of-band edit would.
	mutated := entry.Model
	mutated.UsesPin = !mutated.UsesPin
	data, err := canonicalBytes(mutated)
	if err != nil {
		t.Fatalf("canonicalBytes: %v", err)
	}
	if err := db.Model(&Entity{}).Where("id = ?", id).Update("data", []byte(data)).Error; err != nil {
		t.Fatalf("mutate: %v", err)
	}

	v, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider: %v", err)
	}
	if !v.SeedDrift {
		t.Errorf("SeedDrift = false after mutating the stored document")
	}
	if v.StoredRevision == v.ShippedRevision {
		t.Errorf("revisions are equal after mutation: %q", v.StoredRevision)
	}
	if v.ShippedRevision != entry.Revision {
		t.Errorf("ShippedRevision = %q, want %q - the shipped side must not move", v.ShippedRevision, entry.Revision)
	}
}

// FR-2.4: absence of a shipped file is NOT drift. The template reports an
// empty shippedRevision and seedDrift false.
func TestNoCatalogEntryIsNotDrift(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()

	// A catalog that knows only about the TEST fixture, so GMS 83.1 misses.
	catalog := LoadCatalog(l, filepath.Join("testdata", "templates"))
	p := NewProcessor(l, ctx, db).WithCatalog(catalog)

	id, err := p.Create(createTestRestModel("GMS", 83, 1))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	v, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider: %v", err)
	}
	if v.ShippedRevision != "" {
		t.Errorf("ShippedRevision = %q, want empty", v.ShippedRevision)
	}
	if v.SeedDrift {
		t.Errorf("SeedDrift = true with no shipped file")
	}
	if v.StoredRevision == "" {
		t.Errorf("StoredRevision is empty - it is always computed")
	}
}

// A processor with no catalog wired must degrade to FR-2.4 behaviour rather
// than panicking on a nil map.
func TestUnwiredProcessorReportsNoShippedFile(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	id, err := p.Create(createTestRestModel("GMS", 83, 1))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	v, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider: %v", err)
	}
	if v.ShippedRevision != "" || v.SeedDrift {
		t.Errorf("unwired processor reported shipped=%q drift=%v, want \"\"/false", v.ShippedRevision, v.SeedDrift)
	}
}

// The list and by-region-and-version read paths must carry the same computed
// attributes as the by-id path - the list page renders per-row badges from them.
func TestViewProvidersAgreeAcrossReadPaths(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	catalog := LoadCatalog(l, seedTemplatesDir())

	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}
	p := NewProcessor(l, ctx, db).WithCatalog(catalog)
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	byId, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider: %v", err)
	}
	byVersion, err := p.ViewByRegionAndVersionProvider("GMS", 83, 1)()
	if err != nil {
		t.Fatalf("ViewByRegionAndVersionProvider: %v", err)
	}
	paged, err := p.AllViewProvider(model.Page{Number: 1, Size: 50})()
	if err != nil {
		t.Fatalf("AllViewProvider: %v", err)
	}
	if len(paged.Items) != 1 {
		t.Fatalf("AllViewProvider returned %d items, want 1", len(paged.Items))
	}

	for name, got := range map[string]ViewRestModel{"byVersion": byVersion, "all": paged.Items[0]} {
		if got.ShippedRevision != byId.ShippedRevision || got.StoredRevision != byId.StoredRevision || got.SeedDrift != byId.SeedDrift {
			t.Errorf("%s view disagrees with by-id view: %+v vs %+v", name, got, byId)
		}
	}
}

// D3: the three computed attributes must NOT be persisted. If they ever land
// on RestModel, Create writes them into Entity.Data, Make reads them back, and
// the revision is computed over bytes containing a previous revision.
func TestComputedAttributesAreNotPersisted(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	catalog := LoadCatalog(l, seedTemplatesDir())

	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}
	p := NewProcessor(l, ctx, db).WithCatalog(catalog)
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var e Entity
	if err := db.Where("id = ?", id).First(&e).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	var stored map[string]any
	if err := json.Unmarshal(e.Data, &stored); err != nil {
		t.Fatalf("unmarshal stored document: %v", err)
	}
	for _, k := range []string{"shippedRevision", "storedRevision", "seedDrift"} {
		if _, present := stored[k]; present {
			t.Errorf("computed attribute %q leaked into the stored document", k)
		}
	}
}

// The view model's JSON:API identity must promote from the embedded RestModel,
// or api2go emits the wrong resource type and no id.
func TestViewRestModelIdentityPromotes(t *testing.T) {
	v := ViewRestModel{RestModel: RestModel{Id: "abc"}}
	if v.GetName() != "templates" {
		t.Errorf("GetName() = %q, want templates", v.GetName())
	}
	if v.GetID() != "abc" {
		t.Errorf("GetID() = %q, want abc", v.GetID())
	}
	if err := v.SetID("def"); err != nil {
		t.Fatalf("SetID: %v", err)
	}
	if v.GetID() != "def" {
		t.Errorf("after SetID, GetID() = %q, want def", v.GetID())
	}
}

// encoding/json flattens anonymous embedded structs, so the wire shape is
// RestModel's attributes plus exactly three keys - no per-field restating.
func TestViewRestModelFlattensEmbeddedFields(t *testing.T) {
	v := ViewRestModel{
		RestModel:       createTestRestModel("GMS", 83, 1),
		ShippedRevision: "aa",
		StoredRevision:  "bb",
		SeedDrift:       true,
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, k := range []string{"region", "majorVersion", "minorVersion", "usesPin", "socket", "shippedRevision", "storedRevision", "seedDrift"} {
		if _, present := got[k]; !present {
			t.Errorf("key %q missing from the marshalled view model", k)
		}
	}
	if _, present := got["RestModel"]; present {
		t.Errorf("embedded struct was nested under \"RestModel\" instead of flattened")
	}
}

// FR-3.1 / FR-3.2 / FR-3.3: re-seed restores the shipped content, preserves
// the row's UUID, and leaves the region/version key untouched.
func TestReseedRestoresShippedContent(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	catalog := LoadCatalog(l, seedTemplatesDir())

	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}
	p := NewProcessor(l, ctx, db).WithCatalog(catalog)
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Drift the row.
	mutated := entry.Model
	mutated.UsesPin = !mutated.UsesPin
	data, err := canonicalBytes(mutated)
	if err != nil {
		t.Fatalf("canonicalBytes: %v", err)
	}
	if err := db.Model(&Entity{}).Where("id = ?", id).Update("data", []byte(data)).Error; err != nil {
		t.Fatalf("mutate: %v", err)
	}
	before, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider (before): %v", err)
	}
	if !before.SeedDrift {
		t.Fatalf("precondition failed: template is not drifted")
	}

	if err := p.ReseedById(id); err != nil {
		t.Fatalf("ReseedById: %v", err)
	}

	after, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider (after): %v", err)
	}
	if after.SeedDrift {
		t.Errorf("SeedDrift = true after re-seed")
	}
	if after.StoredRevision != entry.Revision {
		t.Errorf("StoredRevision = %q, want %q", after.StoredRevision, entry.Revision)
	}
	if after.UsesPin != entry.Model.UsesPin {
		t.Errorf("UsesPin = %v, want %v - the mutation survived the re-seed", after.UsesPin, entry.Model.UsesPin)
	}

	var e Entity
	if err := db.Where("id = ?", id).First(&e).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if e.Id != id {
		t.Errorf("Id = %s, want %s - re-seed must never delete-and-recreate", e.Id, id)
	}
	if e.Region != "GMS" || e.MajorVersion != 83 || e.MinorVersion != 1 {
		t.Errorf("key changed: (%s,%d,%d), want (GMS,83,1)", e.Region, e.MajorVersion, e.MinorVersion)
	}
}

// FR-3.7: re-seeding an undrifted template succeeds and leaves the row
// byte-identical.
func TestReseedIsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	catalog := LoadCatalog(l, seedTemplatesDir())

	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}
	p := NewProcessor(l, ctx, db).WithCatalog(catalog)
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var first Entity
	if err := db.Where("id = ?", id).First(&first).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := p.ReseedById(id); err != nil {
			t.Fatalf("ReseedById (call %d): %v", i+1, err)
		}
	}

	var last Entity
	if err := db.Where("id = ?", id).First(&last).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(last.Data) != string(first.Data) {
		t.Errorf("re-seed is not idempotent - Data changed")
	}
}

// The 404 branch: an id with no row.
func TestReseedUnknownIdReturnsNotFound(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db).WithCatalog(LoadCatalog(l, seedTemplatesDir()))

	err := p.ReseedById(uuid.New())
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("error = %v, want ErrTemplateNotFound", err)
	}
}

// The 409 branch: the row exists but no seed file ships for its key.
func TestReseedNoShippedFileReturnsConflict(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()

	// A catalog that knows only the TEST fixture, so GMS 83.1 misses.
	p := NewProcessor(l, ctx, db).WithCatalog(LoadCatalog(l, filepath.Join("testdata", "templates")))
	id, err := p.Create(createTestRestModel("GMS", 83, 1))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = p.ReseedById(id)
	if !errors.Is(err, ErrNoShippedTemplate) {
		t.Errorf("error = %v, want ErrNoShippedTemplate", err)
	}
}

// FR-3.4: re-seed must produce a row byte-identical to a fresh Create of the
// same file - i.e. it must NOT route through UpdateById's preset validator.
func TestReseedProducesSameBytesAsFreshCreate(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	catalog := LoadCatalog(l, seedTemplatesDir())

	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}

	freshDB := setupTestDB(t)
	freshP := NewProcessor(l, ctx, freshDB).WithCatalog(catalog)
	freshId, err := freshP.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create (fresh): %v", err)
	}
	var fresh Entity
	if err := freshDB.Where("id = ?", freshId).First(&fresh).Error; err != nil {
		t.Fatalf("read back (fresh): %v", err)
	}

	reseedDB := setupTestDB(t)
	reseedP := NewProcessor(l, ctx, reseedDB).WithCatalog(catalog)
	reseedId, err := reseedP.Create(createTestRestModel("GMS", 83, 1))
	if err != nil {
		t.Fatalf("Create (stub): %v", err)
	}
	if err := reseedP.ReseedById(reseedId); err != nil {
		t.Fatalf("ReseedById: %v", err)
	}
	var reseeded Entity
	if err := reseedDB.Where("id = ?", reseedId).First(&reseeded).Error; err != nil {
		t.Fatalf("read back (reseeded): %v", err)
	}

	if string(reseeded.Data) != string(fresh.Data) {
		t.Errorf("re-seeded bytes differ from a fresh boot seed of the same file")
	}
}
