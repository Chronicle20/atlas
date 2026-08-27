package history

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestTransform(t *testing.T) {
	loginTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	logoutTime := time.Date(2026, 1, 2, 5, 6, 7, 0, time.UTC)
	m := modelFromEntity(entity{
		ID:          100,
		CharacterId: 200,
		WorldId:     world.Id(1),
		ChannelId:   channel.Id(2),
		LoginTime:   loginTime,
		LogoutTime:  &logoutTime,
	})

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.Id != "100" {
		t.Errorf("Id mismatch. Expected 100, got %v", rm.Id)
	}

	if rm.CharacterId != 200 {
		t.Errorf("CharacterId mismatch. Expected 200, got %v", rm.CharacterId)
	}

	if rm.WorldId != world.Id(1) {
		t.Errorf("WorldId mismatch. Expected %v, got %v", world.Id(1), rm.WorldId)
	}

	if rm.ChannelId != channel.Id(2) {
		t.Errorf("ChannelId mismatch. Expected %v, got %v", channel.Id(2), rm.ChannelId)
	}

	if !rm.LoginTime.Equal(loginTime) {
		t.Errorf("LoginTime mismatch. Expected %v, got %v", loginTime, rm.LoginTime)
	}

	if rm.LogoutTime == nil || !rm.LogoutTime.Equal(logoutTime) {
		t.Errorf("LogoutTime mismatch. Expected %v, got %v", logoutTime, rm.LogoutTime)
	}
}
