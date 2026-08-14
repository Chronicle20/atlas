package pet

import (
	"atlas-pets/pet/exclude"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/stretchr/testify/require"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// patchPetRequest builds a PATCH request carrying a JSON:API document body,
// reusing requestPetsWithTenant (from resource_paginate_test.go, same
// package) for the tenant headers and adding a request body -- PATCH is the
// one method that needs one.
func patchPetRequest(url string, tenantId uuid.UUID, body []byte) *http.Request {
	req := requestPetsWithTenant(http.MethodPatch, url, tenantId)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	return req
}

// TestPatchPetRejectsInvalidName drives PATCH /pets/{petId} through the real
// resource router with a name shorter than petconst's 4-character minimum,
// and expects 400 (not a 500 or a silent no-op write).
func TestPatchPetRejectsInvalidName(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, exclude.Migration, outboxlib.Migration)
	tenantId := uuid.New()
	ctx := databasetest.TenantContext(tenantId)

	pm := seedPet(t, db, ctx, 1, 5000017)

	srv := httptest.NewServer(setupPetRouter(db))
	defer srv.Close()

	body, err := jsonapi.Marshal(RestModel{Id: pm.Id(), Name: "ab"})
	require.NoError(t, err)

	url := fmt.Sprintf("%s/pets/%d", srv.URL, pm.Id())
	req := patchPetRequest(url, tenantId, body)

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// TestPatchPetRenamesPet drives PATCH /pets/{petId} with a valid name and
// confirms the pet is actually renamed -- both in the response body and on
// a subsequent GET -- and that a caller-supplied ownerId in the payload is
// ignored (the owner used for the processor's ownership check comes from the
// stored row, not the request).
func TestPatchPetRenamesPet(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, exclude.Migration, outboxlib.Migration)
	tenantId := uuid.New()
	ctx := databasetest.TenantContext(tenantId)

	pm := seedPet(t, db, ctx, 1, 5000017)

	srv := httptest.NewServer(setupPetRouter(db))
	defer srv.Close()

	// ownerId on the payload deliberately does not match the stored owner (1);
	// if the handler used it instead of the stored row, RenameAndEmit's
	// ownership check would reject the rename.
	body, err := jsonapi.Marshal(RestModel{Id: pm.Id(), Name: "Rexxo", OwnerId: 999})
	require.NoError(t, err)

	url := fmt.Sprintf("%s/pets/%d", srv.URL, pm.Id())
	req := patchPetRequest(url, tenantId, body)

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var respModel RestModel
	require.NoError(t, jsonapi.Unmarshal(respBody, &respModel))
	if respModel.Name != "Rexxo" {
		t.Fatalf("response Name = %q, want %q", respModel.Name, "Rexxo")
	}

	updated, err := NewProcessor(testPaginateLogger(), ctx, db).GetById(pm.Id())
	require.NoError(t, err)
	if updated.Name() != "Rexxo" {
		t.Fatalf("Name() = %q, want %q", updated.Name(), "Rexxo")
	}
	if updated.OwnerId() != 1 {
		t.Fatalf("OwnerId() = %d, want %d (unchanged; payload ownerId must be ignored)", updated.OwnerId(), 1)
	}
}

func TestCreatePetExpiration(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// A zero/epoch expiration (the bare inventory/award path) defaults to
	// now + the standard pet lifespan, so the pet is not created already-expired
	// ("dried up").
	got := createPetExpiration(time.Time{}, now)
	want := now.Add(petLifespan)
	if !got.Equal(want) {
		t.Fatalf("createPetExpiration(zero) = %v, want %v", got, want)
	}
	// A provided (non-zero) expiration is preserved.
	future := now.Add(24 * time.Hour)
	if got := createPetExpiration(future, now); !got.Equal(future) {
		t.Fatalf("createPetExpiration(future) = %v, want %v", got, future)
	}
}

func TestCreatePetName(t *testing.T) {
	// A provided name is preserved.
	if got := createPetName("Fluffy"); got != "Fluffy" {
		t.Fatalf("createPetName(\"Fluffy\") = %q, want %q", got, "Fluffy")
	}
	// An empty name (e.g. a pet granted via the generic inventory/award path,
	// which supplies no name) falls back to "Pet" so the model's "name is
	// required" check passes. The player-facing cash-shop path resolves the WZ
	// name from atlas-data explicitly; the generic award path does not.
	if got := createPetName(""); got != "Pet" {
		t.Fatalf("createPetName(\"\") = %q, want %q", got, "Pet")
	}
}

func TestCreatePetLevel(t *testing.T) {
	// A valid level (1-30) is preserved.
	if got := createPetLevel(15); got != 15 {
		t.Fatalf("createPetLevel(15) = %d, want 15", got)
	}
	// A bare create (level 0, e.g. via the inventory/award path) defaults to 1 so
	// the model's "level must be between 1 and 30" check passes; the processor
	// then applies the rest of the new-pet defaults (closeness 0, full fullness).
	if got := createPetLevel(0); got != 1 {
		t.Fatalf("createPetLevel(0) = %d, want 1", got)
	}
	// Out-of-range high also normalizes to 1.
	if got := createPetLevel(99); got != 1 {
		t.Fatalf("createPetLevel(99) = %d, want 1", got)
	}
}

func TestCreatePetSlot(t *testing.T) {
	// Creation NEVER confers a spawn slot. Slot is a plain int8, so an absent
	// "slot" field decodes as 0 -- which means "spawned in the first pet
	// position". Neither producer sends the field, so every pet was being
	// created already spawned: the client saw a pet it never summoned and could
	// not dismiss, and two purchases both landed in slot 0, a state Spawn itself
	// cannot produce.
	if got := createPetSlot(); got != SlotUnspawned {
		t.Fatalf("createPetSlot() = %d, want %d (unspawned)", got, SlotUnspawned)
	}
	// Guard the constant itself: 0..2 are live pet positions, so anything in
	// that range would mean "spawned".
	if SlotUnspawned >= 0 {
		t.Fatalf("SlotUnspawned = %d, must be negative to mean 'not out'", SlotUnspawned)
	}
}
