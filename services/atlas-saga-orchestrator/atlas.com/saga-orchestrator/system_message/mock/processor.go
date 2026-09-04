package mock

import (
	"atlas-saga-orchestrator/system_message"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
)

// ProcessorMock is a mock implementation of the system_message.Processor interface.
type ProcessorMock struct {
	SendMessageFunc     func(transactionId uuid.UUID, ch channel.Model, characterId uint32, messageType string, message string) error
	PlayPortalSoundFunc func(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	ShowInfoFunc        func(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error
	ShowInfoTextFunc    func(transactionId uuid.UUID, ch channel.Model, characterId uint32, text string) error
	UpdateAreaInfoFunc  func(transactionId uuid.UUID, ch channel.Model, characterId uint32, area uint16, info string) error
	ShowHintFunc        func(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error
	ShowGuideHintFunc   func(transactionId uuid.UUID, ch channel.Model, characterId uint32, hintId uint32, duration uint32) error
	ShowIntroFunc       func(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error
	FieldEffectFunc     func(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error
	UiLockFunc          func(transactionId uuid.UUID, ch channel.Model, characterId uint32, enable bool) error
	UiDisableFunc       func(transactionId uuid.UUID, ch channel.Model, characterId uint32, enable bool) error
	PlaySoundFunc       func(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error
	ChangeMusicFunc     func(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error
	BoatEffectFunc      func(transactionId uuid.UUID, ch channel.Model, characterId uint32, show bool) error
}

var _ system_message.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) SendMessage(transactionId uuid.UUID, ch channel.Model, characterId uint32, messageType string, message string) error {
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(transactionId, ch, characterId, messageType, message)
	}
	return nil
}

func (m *ProcessorMock) PlayPortalSound(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	if m.PlayPortalSoundFunc != nil {
		return m.PlayPortalSoundFunc(transactionId, ch, characterId)
	}
	return nil
}

func (m *ProcessorMock) ShowInfo(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error {
	if m.ShowInfoFunc != nil {
		return m.ShowInfoFunc(transactionId, ch, characterId, path)
	}
	return nil
}

func (m *ProcessorMock) ShowInfoText(transactionId uuid.UUID, ch channel.Model, characterId uint32, text string) error {
	if m.ShowInfoTextFunc != nil {
		return m.ShowInfoTextFunc(transactionId, ch, characterId, text)
	}
	return nil
}

func (m *ProcessorMock) UpdateAreaInfo(transactionId uuid.UUID, ch channel.Model, characterId uint32, area uint16, info string) error {
	if m.UpdateAreaInfoFunc != nil {
		return m.UpdateAreaInfoFunc(transactionId, ch, characterId, area, info)
	}
	return nil
}

func (m *ProcessorMock) ShowHint(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error {
	if m.ShowHintFunc != nil {
		return m.ShowHintFunc(transactionId, ch, characterId, hint, width, height)
	}
	return nil
}

func (m *ProcessorMock) ShowGuideHint(transactionId uuid.UUID, ch channel.Model, characterId uint32, hintId uint32, duration uint32) error {
	if m.ShowGuideHintFunc != nil {
		return m.ShowGuideHintFunc(transactionId, ch, characterId, hintId, duration)
	}
	return nil
}

func (m *ProcessorMock) ShowIntro(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error {
	if m.ShowIntroFunc != nil {
		return m.ShowIntroFunc(transactionId, ch, characterId, path)
	}
	return nil
}

func (m *ProcessorMock) FieldEffect(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error {
	if m.FieldEffectFunc != nil {
		return m.FieldEffectFunc(transactionId, ch, characterId, path)
	}
	return nil
}

func (m *ProcessorMock) UiLock(transactionId uuid.UUID, ch channel.Model, characterId uint32, enable bool) error {
	if m.UiLockFunc != nil {
		return m.UiLockFunc(transactionId, ch, characterId, enable)
	}
	return nil
}

func (m *ProcessorMock) UiDisable(transactionId uuid.UUID, ch channel.Model, characterId uint32, enable bool) error {
	if m.UiDisableFunc != nil {
		return m.UiDisableFunc(transactionId, ch, characterId, enable)
	}
	return nil
}

func (m *ProcessorMock) PlaySound(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error {
	if m.PlaySoundFunc != nil {
		return m.PlaySoundFunc(transactionId, ch, characterId, path)
	}
	return nil
}

func (m *ProcessorMock) ChangeMusic(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error {
	if m.ChangeMusicFunc != nil {
		return m.ChangeMusicFunc(transactionId, ch, characterId, path)
	}
	return nil
}

func (m *ProcessorMock) BoatEffect(transactionId uuid.UUID, ch channel.Model, characterId uint32, show bool) error {
	if m.BoatEffectFunc != nil {
		return m.BoatEffectFunc(transactionId, ch, characterId, show)
	}
	return nil
}
