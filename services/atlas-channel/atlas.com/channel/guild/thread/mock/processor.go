package mock

import (
	"atlas-channel/guild/thread"
)

type ProcessorMock struct {
	GetByIdFunc       func(guildId uint32, threadId uint32) (thread.Model, error)
	GetAllFunc        func(guildId uint32) ([]thread.Model, error)
	ModifyThreadFunc  func(guildId uint32, characterId uint32, threadId uint32, notice bool, title string, message string, emoticonId uint32) error
	CreateThreadFunc  func(guildId uint32, characterId uint32, notice bool, title string, message string, emoticonId uint32) error
	DeleteThreadFunc  func(guildId uint32, characterId uint32, threadId uint32) error
	ListThreadsFunc   func(guildId uint32, characterId uint32, startIndex uint32) error
	ReplyToThreadFunc func(guildId uint32, characterId uint32, threadId uint32, message string) error
	DeleteReplyFunc   func(guildId uint32, characterId uint32, threadId uint32, replyId uint32) error
}

var _ thread.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(guildId uint32, threadId uint32) (thread.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(guildId, threadId)
	}
	return thread.Model{}, nil
}

func (m *ProcessorMock) GetAll(guildId uint32) ([]thread.Model, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(guildId)
	}
	return nil, nil
}

func (m *ProcessorMock) ModifyThread(guildId uint32, characterId uint32, threadId uint32, notice bool, title string, message string, emoticonId uint32) error {
	if m.ModifyThreadFunc != nil {
		return m.ModifyThreadFunc(guildId, characterId, threadId, notice, title, message, emoticonId)
	}
	return nil
}

func (m *ProcessorMock) CreateThread(guildId uint32, characterId uint32, notice bool, title string, message string, emoticonId uint32) error {
	if m.CreateThreadFunc != nil {
		return m.CreateThreadFunc(guildId, characterId, notice, title, message, emoticonId)
	}
	return nil
}

func (m *ProcessorMock) DeleteThread(guildId uint32, characterId uint32, threadId uint32) error {
	if m.DeleteThreadFunc != nil {
		return m.DeleteThreadFunc(guildId, characterId, threadId)
	}
	return nil
}

func (m *ProcessorMock) ListThreads(guildId uint32, characterId uint32, startIndex uint32) error {
	if m.ListThreadsFunc != nil {
		return m.ListThreadsFunc(guildId, characterId, startIndex)
	}
	return nil
}

func (m *ProcessorMock) ReplyToThread(guildId uint32, characterId uint32, threadId uint32, message string) error {
	if m.ReplyToThreadFunc != nil {
		return m.ReplyToThreadFunc(guildId, characterId, threadId, message)
	}
	return nil
}

func (m *ProcessorMock) DeleteReply(guildId uint32, characterId uint32, threadId uint32, replyId uint32) error {
	if m.DeleteReplyFunc != nil {
		return m.DeleteReplyFunc(guildId, characterId, threadId, replyId)
	}
	return nil
}
