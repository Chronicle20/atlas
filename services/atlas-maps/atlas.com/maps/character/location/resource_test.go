package location

import (
	"atlas-maps/data/map/info"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// recordingWarp captures ChangeMap calls.
type recordingWarp struct {
	calls   int
	gotDest field.Model
}

func (r *recordingWarp) ChangeMap(_ uuid.UUID, _ uint32, _ world.Id, dest field.Model, _ uint32, _ bool, _ int16, _ int16) error {
	r.calls++
	r.gotDest = dest
	return nil
}

// erroringWarp always fails ChangeMap, exercising the warp-failure 500 path.
type erroringWarp struct{}

func (erroringWarp) ChangeMap(_ uuid.UUID, _ uint32, _ world.Id, _ field.Model, _ uint32, _ bool, _ int16, _ int16) error {
	return errors.New("warp boom")
}

func TestChangeCharacterLocation_HappyPath(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	lp := NewProcessor(logrus.New(), ctx, db)
	if _, err := lp.Set(7, field.NewBuilder(world.Id(0), 1, _map.Id(100000000)).SetInstance(uuid.Nil).Build()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	ip := &stubInfoProcessor{out: info.NewBuilder().SetId(104000000).Build()} // err nil ⇒ map exists
	rw := &recordingWarp{}

	status, err := changeCharacterLocation(logrus.New(), lp, ip, rw, 7, _map.Id(104000000))
	if status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", status)
	}
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if rw.calls != 1 {
		t.Fatalf("ChangeMap calls = %d, want 1", rw.calls)
	}
	if rw.gotDest.MapId() != _map.Id(104000000) || rw.gotDest.ChannelId() != 1 || rw.gotDest.Instance() != uuid.Nil {
		t.Fatalf("dest mismatch: %+v", rw.gotDest)
	}
}

func TestChangeCharacterLocation_InvalidMap_400_NoWarp(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	lp := NewProcessor(logrus.New(), ctx, db)
	if _, err := lp.Set(7, field.NewBuilder(world.Id(0), 1, _map.Id(100000000)).SetInstance(uuid.Nil).Build()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	ip := &stubInfoProcessor{err: requests.ErrNotFound}
	rw := &recordingWarp{}

	status, err := changeCharacterLocation(logrus.New(), lp, ip, rw, 7, _map.Id(999999999))
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if rw.calls != 0 {
		t.Fatalf("ChangeMap must not be called on invalid map; got %d calls", rw.calls)
	}
}

func TestChangeCharacterLocation_MapCheckInfraError_500(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	lp := NewProcessor(logrus.New(), ctx, db)
	if _, err := lp.Set(7, field.NewBuilder(world.Id(0), 1, _map.Id(100000000)).SetInstance(uuid.Nil).Build()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	ip := &stubInfoProcessor{err: errors.New("boom")} // non-ErrNotFound ⇒ infra failure
	rw := &recordingWarp{}

	status, err := changeCharacterLocation(logrus.New(), lp, ip, rw, 7, _map.Id(104000000))
	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", status)
	}
	if err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
	if rw.calls != 0 {
		t.Fatalf("ChangeMap must not be called when map check fails for infra reasons; got %d calls", rw.calls)
	}
}

func TestChangeCharacterLocation_WarpError_500(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	lp := NewProcessor(logrus.New(), ctx, db)
	if _, err := lp.Set(7, field.NewBuilder(world.Id(0), 1, _map.Id(100000000)).SetInstance(uuid.Nil).Build()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	ip := &stubInfoProcessor{out: info.NewBuilder().SetId(104000000).Build()} // map exists
	rw := erroringWarp{}

	status, err := changeCharacterLocation(logrus.New(), lp, ip, rw, 7, _map.Id(104000000))
	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", status)
	}
	if err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
}

func TestChangeCharacterLocation_NoRow_404(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	lp := NewProcessor(logrus.New(), ctx, db)
	ip := &stubInfoProcessor{out: info.NewBuilder().SetId(104000000).Build()}
	rw := &recordingWarp{}

	status, err := changeCharacterLocation(logrus.New(), lp, ip, rw, 7, _map.Id(104000000))
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if rw.calls != 0 {
		t.Fatalf("ChangeMap must not be called when no row; got %d calls", rw.calls)
	}
}

// TestTransform_CarriesState proves the discriminator survives the projection
// atlas-channel's /find path reads.
func TestTransform_CarriesState(t *testing.T) {
	m := NewBuilder(1234).
		SetWorldId(world.Id(0)).
		SetChannelId(channel.Id(7)).
		SetMapId(_map.Id(100000000)).
		SetInstance(uuid.Nil).
		SetState(characterconst.PresenceStateInCashShop).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if rm.State != characterconst.PresenceStateInCashShop {
		t.Errorf("State = %q, want IN_CASH_SHOP", rm.State)
	}
	if rm.ChannelId != channel.Id(7) {
		t.Errorf("ChannelId = %d, want 7", rm.ChannelId)
	}
	if rm.GetName() != "character-locations" {
		t.Errorf("GetName = %q, want character-locations", rm.GetName())
	}
}

// TestRestModel_StateJSONKey pins the wire key itself. atlas-channel decodes
// "state"; renaming or re-casing it silently degrades /find to "not findable"
// on every off-channel target, with no error anywhere.
func TestRestModel_StateJSONKey(t *testing.T) {
	rm := RestModel{
		Id:        1234,
		WorldId:   world.Id(0),
		ChannelId: channel.Id(7),
		MapId:     _map.Id(100000000),
		Instance:  uuid.Nil,
		State:     characterconst.PresenceStateInField,
	}

	b, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	got, ok := decoded["state"]
	if !ok {
		t.Fatalf("no state key in %s", string(b))
	}
	if got != "IN_FIELD" {
		t.Errorf("state = %v, want IN_FIELD", got)
	}
	// The pre-existing attributes must still be there — this is an additive
	// change, and atlas-character / atlas-login / the session bootstrap all
	// read them.
	for _, k := range []string{"worldId", "channelId", "mapId", "instance"} {
		if _, ok := decoded[k]; !ok {
			t.Errorf("attribute %q disappeared from %s", k, string(b))
		}
	}
}
