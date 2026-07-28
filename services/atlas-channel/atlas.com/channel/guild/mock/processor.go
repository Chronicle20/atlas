package mock

import (
	"atlas-channel/guild"
	"atlas-channel/guild/member"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	GetByIdFunc                  func(guildId uint32) (guild.Model, error)
	GetByMemberIdFunc            func(memberId uint32) (guild.Model, error)
	ByMemberIdProviderFunc       func(memberId uint32) model.Provider[[]guild.Model]
	GetMemberIdsFunc             func(guildId uint32, filters []model.Filter[member.Model]) model.Provider[[]uint32]
	RequestCreateFunc            func(f field.Model, characterId uint32, name string) error
	CreationAgreementFunc        func(characterId uint32, agreed bool) error
	RequestEmblemUpdateFunc      func(guildId uint32, characterId uint32, logoBackground uint16, logoBackgroundColor byte, logo uint16, logoColor byte) error
	RequestNoticeUpdateFunc      func(guildId uint32, characterId uint32, notice string) error
	LeaveFunc                    func(guildId uint32, characterId uint32) error
	ExpelFunc                    func(guildId uint32, characterId uint32, targetId uint32, targetName string) error
	RequestInviteFunc            func(guildId uint32, characterId uint32, targetId uint32) error
	RequestTitleChangesFunc      func(guildId uint32, characterId uint32, titles []string) error
	RequestMemberTitleUpdateFunc func(guildId uint32, characterId uint32, targetId uint32, newTitle byte) error
}

var _ guild.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(guildId uint32) (guild.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(guildId)
	}
	return guild.Model{}, nil
}

func (m *ProcessorMock) GetByMemberId(memberId uint32) (guild.Model, error) {
	if m.GetByMemberIdFunc != nil {
		return m.GetByMemberIdFunc(memberId)
	}
	return guild.Model{}, nil
}

func (m *ProcessorMock) ByMemberIdProvider(memberId uint32) model.Provider[[]guild.Model] {
	if m.ByMemberIdProviderFunc != nil {
		return m.ByMemberIdProviderFunc(memberId)
	}
	return model.FixedProvider([]guild.Model{})
}

func (m *ProcessorMock) GetMemberIds(guildId uint32, filters []model.Filter[member.Model]) model.Provider[[]uint32] {
	if m.GetMemberIdsFunc != nil {
		return m.GetMemberIdsFunc(guildId, filters)
	}
	return model.FixedProvider([]uint32{})
}

func (m *ProcessorMock) RequestCreate(f field.Model, characterId uint32, name string) error {
	if m.RequestCreateFunc != nil {
		return m.RequestCreateFunc(f, characterId, name)
	}
	return nil
}

func (m *ProcessorMock) CreationAgreement(characterId uint32, agreed bool) error {
	if m.CreationAgreementFunc != nil {
		return m.CreationAgreementFunc(characterId, agreed)
	}
	return nil
}

func (m *ProcessorMock) RequestEmblemUpdate(guildId uint32, characterId uint32, logoBackground uint16, logoBackgroundColor byte, logo uint16, logoColor byte) error {
	if m.RequestEmblemUpdateFunc != nil {
		return m.RequestEmblemUpdateFunc(guildId, characterId, logoBackground, logoBackgroundColor, logo, logoColor)
	}
	return nil
}

func (m *ProcessorMock) RequestNoticeUpdate(guildId uint32, characterId uint32, notice string) error {
	if m.RequestNoticeUpdateFunc != nil {
		return m.RequestNoticeUpdateFunc(guildId, characterId, notice)
	}
	return nil
}

func (m *ProcessorMock) Leave(guildId uint32, characterId uint32) error {
	if m.LeaveFunc != nil {
		return m.LeaveFunc(guildId, characterId)
	}
	return nil
}

func (m *ProcessorMock) Expel(guildId uint32, characterId uint32, targetId uint32, targetName string) error {
	if m.ExpelFunc != nil {
		return m.ExpelFunc(guildId, characterId, targetId, targetName)
	}
	return nil
}

func (m *ProcessorMock) RequestInvite(guildId uint32, characterId uint32, targetId uint32) error {
	if m.RequestInviteFunc != nil {
		return m.RequestInviteFunc(guildId, characterId, targetId)
	}
	return nil
}

func (m *ProcessorMock) RequestTitleChanges(guildId uint32, characterId uint32, titles []string) error {
	if m.RequestTitleChangesFunc != nil {
		return m.RequestTitleChangesFunc(guildId, characterId, titles)
	}
	return nil
}

func (m *ProcessorMock) RequestMemberTitleUpdate(guildId uint32, characterId uint32, targetId uint32, newTitle byte) error {
	if m.RequestMemberTitleUpdateFunc != nil {
		return m.RequestMemberTitleUpdateFunc(guildId, characterId, targetId, newTitle)
	}
	return nil
}
