package location

import (
	"net/http"
	"testing"

	"atlas-maps/data/map/info"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// recordingWarp captures ChangeMap calls.
type recordingWarp struct {
	calls   int
	gotDest field.Model
}

func (r *recordingWarp) ChangeMap(_ uuid.UUID, _ uint32, _ world.Id, dest field.Model, _ uint32) error {
	r.calls++
	r.gotDest = dest
	return nil
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

	status := changeCharacterLocation(logrus.New(), lp, ip, rw, 7, _map.Id(104000000))
	if status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", status)
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

	status := changeCharacterLocation(logrus.New(), lp, ip, rw, 7, _map.Id(999999999))
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
	if rw.calls != 0 {
		t.Fatalf("ChangeMap must not be called on invalid map; got %d calls", rw.calls)
	}
}

func TestChangeCharacterLocation_NoRow_404(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	lp := NewProcessor(logrus.New(), ctx, db)
	ip := &stubInfoProcessor{out: info.NewBuilder().SetId(104000000).Build()}
	rw := &recordingWarp{}

	status := changeCharacterLocation(logrus.New(), lp, ip, rw, 7, _map.Id(104000000))
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
	if rw.calls != 0 {
		t.Fatalf("ChangeMap must not be called when no row; got %d calls", rw.calls)
	}
}
