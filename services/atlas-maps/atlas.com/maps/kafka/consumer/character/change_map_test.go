package character

import (
	"atlas-maps/character/warp"
	"testing"

	characterKafka "atlas-maps/kafka/message/character"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type recordingWarp struct {
	gotDest        field.Model
	gotPortalId    uint32
	gotUsePosition bool
	gotX           int16
	gotY           int16
	calls          int
}

func (r *recordingWarp) ChangeMap(_ uuid.UUID, _ uint32, _ world.Id, dest field.Model, portalId uint32, useTargetPosition bool, targetX int16, targetY int16) error {
	r.calls++
	r.gotDest = dest
	r.gotPortalId = portalId
	r.gotUsePosition = useTargetPosition
	r.gotX = targetX
	r.gotY = targetY
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

func TestChangeMapFromCommand_ThreadsTargetPosition(t *testing.T) {
	rw := &recordingWarp{}
	cmd := characterKafka.Command[characterKafka.ChangeMapBody]{
		WorldId:     1,
		CharacterId: 999,
		Type:        characterKafka.CommandChangeMap,
		Body: characterKafka.ChangeMapBody{
			ChannelId:         2,
			MapId:             _map.Id(240000000),
			Instance:          uuid.New(),
			UseTargetPosition: true,
			TargetX:           123,
			TargetY:           -456,
		},
	}
	if err := changeMapFromCommand(warp.Processor(rw))(cmd); err != nil {
		t.Fatalf("changeMapFromCommand: %v", err)
	}
	if !rw.gotUsePosition || rw.gotX != 123 || rw.gotY != -456 {
		t.Fatalf("position not threaded: usePos=%v x=%d y=%d", rw.gotUsePosition, rw.gotX, rw.gotY)
	}
}
