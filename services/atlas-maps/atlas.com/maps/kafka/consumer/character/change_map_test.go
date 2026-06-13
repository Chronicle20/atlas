package character

import (
	"testing"

	"atlas-maps/character/warp"
	characterKafka "atlas-maps/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

type recordingWarp struct {
	gotDest     field.Model
	gotPortalId uint32
	calls       int
}

func (r *recordingWarp) ChangeMap(_ uuid.UUID, _ uint32, _ world.Id, dest field.Model, portalId uint32) error {
	r.calls++
	r.gotDest = dest
	r.gotPortalId = portalId
	return nil
}

func TestChangeMapFromCommand_FunnelsThroughWarp(t *testing.T) {
	rw := &recordingWarp{}
	inst := uuid.New()
	cmd := characterKafka.Command[characterKafka.ChangeMapBody]{
		WorldId:     1,
		CharacterId: 999,
		Type:        characterKafka.CommandChangeMap,
		Body: characterKafka.ChangeMapBody{
			ChannelId: 2,
			MapId:     _map.Id(240000000),
			Instance:  inst,
			PortalId:  3,
		},
	}
	if err := changeMapFromCommand(warp.Processor(rw))(cmd); err != nil {
		t.Fatalf("changeMapFromCommand: %v", err)
	}
	if rw.calls != 1 {
		t.Fatalf("ChangeMap called %d times, want 1", rw.calls)
	}
	if rw.gotDest.MapId() != _map.Id(240000000) || rw.gotDest.ChannelId() != 2 || rw.gotDest.Instance() != inst {
		t.Fatalf("dest mismatch: %+v", rw.gotDest)
	}
	if rw.gotPortalId != 3 {
		t.Fatalf("portalId = %d, want 3", rw.gotPortalId)
	}
}
