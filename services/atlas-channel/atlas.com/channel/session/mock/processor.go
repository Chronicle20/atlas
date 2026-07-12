package mock

import (
	"atlas-channel/session"
	"context"
	"net"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	WithContextFunc                func(ctx context.Context) session.Processor
	AllInTenantProviderFunc        func() ([]session.Model, error)
	AllInChannelProviderFunc       func(worldId world.Id, channelId channel.Id) ([]session.Model, error)
	ByIdModelProviderFunc          func(sessionId uuid.UUID) model.Provider[session.Model]
	IfPresentByIdFunc              func(sessionId uuid.UUID, f model.Operator[session.Model])
	IfPresentByIdInWorldFunc       func(sessionId uuid.UUID, ch channel.Model, f model.Operator[session.Model])
	ByCharacterIdModelProviderFunc func(ch channel.Model) func(characterId uint32) model.Provider[session.Model]
	IfPresentByCharacterIdFunc     func(ch channel.Model) func(characterId uint32, f model.Operator[session.Model]) error
	ByAccountIdModelProviderFunc   func(ch channel.Model) func(accountId uint32) model.Provider[session.Model]
	IfPresentByAccountIdFunc       func(ch channel.Model) func(accountId uint32, f model.Operator[session.Model]) error
	GetByCharacterIdFunc           func(ch channel.Model) func(characterId uint32) (session.Model, error)
	ForEachByCharacterIdFunc       func(ch channel.Model) func(provider model.Provider[[]uint32], f model.Operator[session.Model]) error
	SetAccountIdFunc               func(id uuid.UUID, accountId uint32) session.Model
	SetCharacterIdFunc             func(id uuid.UUID, characterId uint32) session.Model
	SetMapIdFunc                   func(id uuid.UUID, mapId _map.Id) session.Model
	SetFieldFunc                   func(id uuid.UUID, f field.Model) session.Model
	SetGmFunc                      func(id uuid.UUID, gm bool) session.Model
	UpdateLastRequestFunc          func(id uuid.UUID) session.Model
	SessionCreatedFunc             func(s session.Model) error
	CreateFunc                     func(ch channel.Model, locale byte) func(sessionId uuid.UUID, conn net.Conn)
	DecryptFunc                    func(hasAes bool, hasMapleEncryption bool) func(sessionId uuid.UUID, input []byte) []byte
	DestroyByIdWithSpanFunc        func(sessionId uuid.UUID)
	DestroyByIdFunc                func(sessionId uuid.UUID)
	DestroyFunc                    func(s session.Model) error
	SetStorageNpcIdFunc            func(id uuid.UUID, npcId uint32) session.Model
	ClearStorageNpcIdFunc          func(id uuid.UUID) session.Model
}

var _ session.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) WithContext(ctx context.Context) session.Processor {
	if m.WithContextFunc != nil {
		return m.WithContextFunc(ctx)
	}
	return m
}

func (m *ProcessorMock) AllInTenantProvider() ([]session.Model, error) {
	if m.AllInTenantProviderFunc != nil {
		return m.AllInTenantProviderFunc()
	}
	return nil, nil
}

func (m *ProcessorMock) AllInChannelProvider(worldId world.Id, channelId channel.Id) ([]session.Model, error) {
	if m.AllInChannelProviderFunc != nil {
		return m.AllInChannelProviderFunc(worldId, channelId)
	}
	return nil, nil
}

func (m *ProcessorMock) ByIdModelProvider(sessionId uuid.UUID) model.Provider[session.Model] {
	if m.ByIdModelProviderFunc != nil {
		return m.ByIdModelProviderFunc(sessionId)
	}
	return model.FixedProvider(session.Model{})
}

func (m *ProcessorMock) IfPresentById(sessionId uuid.UUID, f model.Operator[session.Model]) {
	if m.IfPresentByIdFunc != nil {
		m.IfPresentByIdFunc(sessionId, f)
	}
}

func (m *ProcessorMock) IfPresentByIdInWorld(sessionId uuid.UUID, ch channel.Model, f model.Operator[session.Model]) {
	if m.IfPresentByIdInWorldFunc != nil {
		m.IfPresentByIdInWorldFunc(sessionId, ch, f)
	}
}

func (m *ProcessorMock) ByCharacterIdModelProvider(ch channel.Model) func(characterId uint32) model.Provider[session.Model] {
	if m.ByCharacterIdModelProviderFunc != nil {
		return m.ByCharacterIdModelProviderFunc(ch)
	}
	return func(characterId uint32) model.Provider[session.Model] {
		return model.FixedProvider(session.Model{})
	}
}

func (m *ProcessorMock) IfPresentByCharacterId(ch channel.Model) func(characterId uint32, f model.Operator[session.Model]) error {
	if m.IfPresentByCharacterIdFunc != nil {
		return m.IfPresentByCharacterIdFunc(ch)
	}
	return func(characterId uint32, f model.Operator[session.Model]) error {
		return nil
	}
}

func (m *ProcessorMock) ByAccountIdModelProvider(ch channel.Model) func(accountId uint32) model.Provider[session.Model] {
	if m.ByAccountIdModelProviderFunc != nil {
		return m.ByAccountIdModelProviderFunc(ch)
	}
	return func(accountId uint32) model.Provider[session.Model] {
		return model.FixedProvider(session.Model{})
	}
}

func (m *ProcessorMock) IfPresentByAccountId(ch channel.Model) func(accountId uint32, f model.Operator[session.Model]) error {
	if m.IfPresentByAccountIdFunc != nil {
		return m.IfPresentByAccountIdFunc(ch)
	}
	return func(accountId uint32, f model.Operator[session.Model]) error {
		return nil
	}
}

func (m *ProcessorMock) GetByCharacterId(ch channel.Model) func(characterId uint32) (session.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(ch)
	}
	return func(characterId uint32) (session.Model, error) {
		return session.Model{}, nil
	}
}

func (m *ProcessorMock) ForEachByCharacterId(ch channel.Model) func(provider model.Provider[[]uint32], f model.Operator[session.Model]) error {
	if m.ForEachByCharacterIdFunc != nil {
		return m.ForEachByCharacterIdFunc(ch)
	}
	return func(provider model.Provider[[]uint32], f model.Operator[session.Model]) error {
		return nil
	}
}

func (m *ProcessorMock) SetAccountId(id uuid.UUID, accountId uint32) session.Model {
	if m.SetAccountIdFunc != nil {
		return m.SetAccountIdFunc(id, accountId)
	}
	return session.Model{}
}

func (m *ProcessorMock) SetCharacterId(id uuid.UUID, characterId uint32) session.Model {
	if m.SetCharacterIdFunc != nil {
		return m.SetCharacterIdFunc(id, characterId)
	}
	return session.Model{}
}

func (m *ProcessorMock) SetMapId(id uuid.UUID, mapId _map.Id) session.Model {
	if m.SetMapIdFunc != nil {
		return m.SetMapIdFunc(id, mapId)
	}
	return session.Model{}
}

func (m *ProcessorMock) SetField(id uuid.UUID, f field.Model) session.Model {
	if m.SetFieldFunc != nil {
		return m.SetFieldFunc(id, f)
	}
	return session.Model{}
}

func (m *ProcessorMock) SetGm(id uuid.UUID, gm bool) session.Model {
	if m.SetGmFunc != nil {
		return m.SetGmFunc(id, gm)
	}
	return session.Model{}
}

func (m *ProcessorMock) UpdateLastRequest(id uuid.UUID) session.Model {
	if m.UpdateLastRequestFunc != nil {
		return m.UpdateLastRequestFunc(id)
	}
	return session.Model{}
}

func (m *ProcessorMock) SessionCreated(s session.Model) error {
	if m.SessionCreatedFunc != nil {
		return m.SessionCreatedFunc(s)
	}
	return nil
}

func (m *ProcessorMock) Create(ch channel.Model, locale byte) func(sessionId uuid.UUID, conn net.Conn) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ch, locale)
	}
	return func(sessionId uuid.UUID, conn net.Conn) {}
}

func (m *ProcessorMock) Decrypt(hasAes bool, hasMapleEncryption bool) func(sessionId uuid.UUID, input []byte) []byte {
	if m.DecryptFunc != nil {
		return m.DecryptFunc(hasAes, hasMapleEncryption)
	}
	return func(sessionId uuid.UUID, input []byte) []byte {
		return nil
	}
}

func (m *ProcessorMock) DestroyByIdWithSpan(sessionId uuid.UUID) {
	if m.DestroyByIdWithSpanFunc != nil {
		m.DestroyByIdWithSpanFunc(sessionId)
	}
}

func (m *ProcessorMock) DestroyById(sessionId uuid.UUID) {
	if m.DestroyByIdFunc != nil {
		m.DestroyByIdFunc(sessionId)
	}
}

func (m *ProcessorMock) Destroy(s session.Model) error {
	if m.DestroyFunc != nil {
		return m.DestroyFunc(s)
	}
	return nil
}

func (m *ProcessorMock) SetStorageNpcId(id uuid.UUID, npcId uint32) session.Model {
	if m.SetStorageNpcIdFunc != nil {
		return m.SetStorageNpcIdFunc(id, npcId)
	}
	return session.Model{}
}

func (m *ProcessorMock) ClearStorageNpcId(id uuid.UUID) session.Model {
	if m.ClearStorageNpcIdFunc != nil {
		return m.ClearStorageNpcIdFunc(id)
	}
	return session.Model{}
}
