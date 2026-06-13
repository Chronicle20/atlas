package character

import (
	charmsg "atlas-mounts/kafka/message/character"
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// fakeSeam swaps the registryRemove seam for an in-memory recorder and restores
// it when the returned cleanup runs.
type fakeSeam struct {
	removeCalls []uint32
}

func newFake(t *testing.T) *fakeSeam {
	t.Helper()
	f := &fakeSeam{}

	orig := registryRemove
	registryRemove = func(_ context.Context, characterId uint32) error {
		f.removeCalls = append(f.removeCalls, characterId)
		return nil
	}
	t.Cleanup(func() { registryRemove = orig })
	return f
}

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

func TestHandleStatusEvent_Logout(t *testing.T) {
	f := newFake(t)

	e := charmsg.StatusEvent[any]{
		CharacterId: 1000,
		Type:        charmsg.StatusEventTypeLogout,
	}

	handleStatusEvent(testLogger(), context.Background(), e)

	assert.Equal(t, []uint32{1000}, f.removeCalls, "logout must remove the active-mount registry entry once")
}

func TestHandleStatusEvent_Login(t *testing.T) {
	f := newFake(t)

	e := charmsg.StatusEvent[any]{
		CharacterId: 2000,
		Type:        charmsg.StatusEventTypeLogin,
	}

	handleStatusEvent(testLogger(), context.Background(), e)

	assert.Empty(t, f.removeCalls, "login must NOT touch the registry (no-op)")
}

func TestHandleStatusEvent_OtherType(t *testing.T) {
	f := newFake(t)

	e := charmsg.StatusEvent[any]{
		CharacterId: 3000,
		Type:        "MAP_CHANGED",
	}

	handleStatusEvent(testLogger(), context.Background(), e)

	assert.Empty(t, f.removeCalls, "non-logout status events must not touch the registry")
}
