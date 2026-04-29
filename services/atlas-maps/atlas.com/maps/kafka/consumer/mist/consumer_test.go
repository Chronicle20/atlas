package mist

import (
	mistKafka "atlas-maps/kafka/message/mist"
	mistDomain "atlas-maps/mist"
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// fakeProcessor records calls to Create and Destroy so we can assert the
// consumer dispatches commands to the right method without exercising the
// real registry/producer plumbing.
type fakeProcessor struct {
	mu          sync.Mutex
	createCalls []mistKafka.CreateCommandBody
	destroyCalls []destroyCall
	createErr   error
	destroyErr  error
}

type destroyCall struct {
	id     uuid.UUID
	reason string
}

func (f *fakeProcessor) Create(body mistKafka.CreateCommandBody) (mistDomain.Mist, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls = append(f.createCalls, body)
	if f.createErr != nil {
		return mistDomain.Mist{}, f.createErr
	}
	return mistDomain.Mist{}, nil
}

func (f *fakeProcessor) Destroy(id uuid.UUID, reason string) (mistDomain.Mist, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.destroyCalls = append(f.destroyCalls, destroyCall{id: id, reason: reason})
	if f.destroyErr != nil {
		return mistDomain.Mist{}, f.destroyErr
	}
	return mistDomain.Mist{}, nil
}

// installFakeProcessor swaps the package-level processorFactory for one that
// always returns the given fake. The returned cleanup restores the original.
func installFakeProcessor(t *testing.T, fp *fakeProcessor) func() {
	t.Helper()
	original := processorFactory
	processorFactory = func(_ logrus.FieldLogger, _ context.Context) mistDomain.Processor {
		return fp
	}
	return func() { processorFactory = original }
}

func TestHandleCreateCommand_DispatchesToProcessor(t *testing.T) {
	fp := &fakeProcessor{}
	cleanup := installFakeProcessor(t, fp)
	defer cleanup()

	logger, _ := test.NewNullLogger()
	body := mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType: "MONSTER", OwnerId: 9001,
		OriginX: 100, OriginY: 200,
		LtX: -50, LtY: -30, RbX: 50, RbY: 30,
		Disease: "POISON", DiseaseValue: 80, DiseaseDuration: 30000,
		Duration: 10000, TickIntervalMs: 1000,
		SourceSkillId: 100020, SourceSkillLevel: 5,
	}
	cmd := mistKafka.Command[mistKafka.CreateCommandBody]{
		Tenant: uuid.New(),
		Type:   mistKafka.CommandTypeCreate,
		Body:   body,
	}

	handleCreateCommand()(logger, context.Background(), cmd)

	require.Len(t, fp.createCalls, 1, "expected exactly one Create call")
	require.Equal(t, body, fp.createCalls[0])
	require.Empty(t, fp.destroyCalls)
}

func TestHandleCreateCommand_SkipsWrongType(t *testing.T) {
	fp := &fakeProcessor{}
	cleanup := installFakeProcessor(t, fp)
	defer cleanup()

	logger, _ := test.NewNullLogger()
	cmd := mistKafka.Command[mistKafka.CreateCommandBody]{
		Tenant: uuid.New(),
		Type:   "SOMETHING_ELSE",
		Body:   mistKafka.CreateCommandBody{Duration: 1000},
	}

	handleCreateCommand()(logger, context.Background(), cmd)

	require.Empty(t, fp.createCalls)
}

func TestHandleCreateCommand_ErrorIsLoggedNotPanicked(t *testing.T) {
	fp := &fakeProcessor{createErr: errors.New("boom")}
	cleanup := installFakeProcessor(t, fp)
	defer cleanup()

	logger, _ := test.NewNullLogger()
	cmd := mistKafka.Command[mistKafka.CreateCommandBody]{
		Type: mistKafka.CommandTypeCreate,
		Body: mistKafka.CreateCommandBody{Duration: 1000},
	}

	require.NotPanics(t, func() {
		handleCreateCommand()(logger, context.Background(), cmd)
	})
	require.Len(t, fp.createCalls, 1)
}

func TestHandleCancelCommand_DispatchesWithCancelledReason(t *testing.T) {
	fp := &fakeProcessor{}
	cleanup := installFakeProcessor(t, fp)
	defer cleanup()

	logger, _ := test.NewNullLogger()
	mistId := uuid.New()
	cmd := mistKafka.Command[mistKafka.CancelCommandBody]{
		Tenant: uuid.New(),
		Type:   mistKafka.CommandTypeCancel,
		Body:   mistKafka.CancelCommandBody{MistId: mistId},
	}

	handleCancelCommand()(logger, context.Background(), cmd)

	require.Len(t, fp.destroyCalls, 1, "expected exactly one Destroy call")
	require.Equal(t, mistId, fp.destroyCalls[0].id)
	require.Equal(t, mistKafka.ReasonCancelled, fp.destroyCalls[0].reason,
		"cancel command must map to CANCELLED reason")
	require.Empty(t, fp.createCalls)
}

func TestHandleCancelCommand_SkipsWrongType(t *testing.T) {
	fp := &fakeProcessor{}
	cleanup := installFakeProcessor(t, fp)
	defer cleanup()

	logger, _ := test.NewNullLogger()
	cmd := mistKafka.Command[mistKafka.CancelCommandBody]{
		Type: "SOMETHING_ELSE",
		Body: mistKafka.CancelCommandBody{MistId: uuid.New()},
	}

	handleCancelCommand()(logger, context.Background(), cmd)

	require.Empty(t, fp.destroyCalls)
}

func TestHandleCancelCommand_ErrorIsLoggedNotPanicked(t *testing.T) {
	fp := &fakeProcessor{destroyErr: errors.New("not found")}
	cleanup := installFakeProcessor(t, fp)
	defer cleanup()

	logger, _ := test.NewNullLogger()
	cmd := mistKafka.Command[mistKafka.CancelCommandBody]{
		Type: mistKafka.CommandTypeCancel,
		Body: mistKafka.CancelCommandBody{MistId: uuid.New()},
	}

	require.NotPanics(t, func() {
		handleCancelCommand()(logger, context.Background(), cmd)
	})
	require.Len(t, fp.destroyCalls, 1)
}
