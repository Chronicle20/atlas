package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const WhisperWriter = "CharacterChatWhisper"

// WhisperSendResult - result of attempting to send a whisper
type WhisperSendResult struct {
	mode       byte
	targetName string
	success    bool
}

func NewWhisperSendResult(mode byte, targetName string, success bool) WhisperSendResult {
	return WhisperSendResult{mode: mode, targetName: targetName, success: success}
}

func (m WhisperSendResult) Mode() byte        { return m.mode }
func (m WhisperSendResult) TargetName() string { return m.targetName }
func (m WhisperSendResult) Success() bool      { return m.success }

func (m WhisperSendResult) Operation() string { return WhisperWriter }
func (m WhisperSendResult) String() string {
	return fmt.Sprintf("whisper send result to [%s] success [%t]", m.targetName, m.success)
}

func (m WhisperSendResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.targetName)
		w.WriteBool(m.success)
		return w.Bytes()
	}
}

func (m *WhisperSendResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.targetName = r.ReadAsciiString()
		m.success = r.ReadBool()
	}
}

// WhisperReceive - incoming whisper from another character
type WhisperReceive struct {
	mode      byte
	fromName  string
	channelId byte
	gm        bool
	message   string
}

func NewWhisperReceive(mode byte, fromName string, channelId byte, gm bool, message string) WhisperReceive {
	return WhisperReceive{mode: mode, fromName: fromName, channelId: channelId, gm: gm, message: message}
}

func (m WhisperReceive) Mode() byte       { return m.mode }
func (m WhisperReceive) FromName() string  { return m.fromName }
func (m WhisperReceive) ChannelId() byte   { return m.channelId }
func (m WhisperReceive) Gm() bool          { return m.gm }
func (m WhisperReceive) Message() string   { return m.message }

func (m WhisperReceive) Operation() string { return WhisperWriter }
func (m WhisperReceive) String() string {
	return fmt.Sprintf("whisper receive from [%s]", m.fromName)
}

func (m WhisperReceive) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.fromName)
		w.WriteByte(m.channelId)
		w.WriteBool(m.gm)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *WhisperReceive) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.fromName = r.ReadAsciiString()
		m.channelId = r.ReadByte()
		m.gm = r.ReadBool()
		m.message = r.ReadAsciiString()
	}
}

// WhisperFindResultCashShop - target is in cash shop
type WhisperFindResultCashShop struct {
	mode       byte
	targetName string
}

func NewWhisperFindResultCashShop(mode byte, targetName string) WhisperFindResultCashShop {
	return WhisperFindResultCashShop{mode: mode, targetName: targetName}
}

func (m WhisperFindResultCashShop) Mode() byte        { return m.mode }
func (m WhisperFindResultCashShop) TargetName() string { return m.targetName }

func (m WhisperFindResultCashShop) Operation() string { return WhisperWriter }
func (m WhisperFindResultCashShop) String() string {
	return fmt.Sprintf("whisper find [%s] in cash shop", m.targetName)
}

func (m WhisperFindResultCashShop) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.targetName)
		w.WriteByte(2)
		w.WriteInt32(-1)
		return w.Bytes()
	}
}

func (m *WhisperFindResultCashShop) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.targetName = r.ReadAsciiString()
		_ = r.ReadByte()  // findMode = 2
		_ = r.ReadInt32() // -1
	}
}

// WhisperFindResultMap - target is on a map; optionally includes x/y coordinates
type WhisperFindResultMap struct {
	mode       byte
	targetName string
	mapId      uint32
	includeXY  bool
	x          int16
	y          int16
}

func NewWhisperFindResultMap(mode byte, targetName string, mapId uint32) WhisperFindResultMap {
	return WhisperFindResultMap{mode: mode, targetName: targetName, mapId: mapId}
}

func NewWhisperFindResultMapWithXY(mode byte, targetName string, mapId uint32, x int16, y int16) WhisperFindResultMap {
	return WhisperFindResultMap{mode: mode, targetName: targetName, mapId: mapId, includeXY: true, x: x, y: y}
}

func (m WhisperFindResultMap) Mode() byte        { return m.mode }
func (m WhisperFindResultMap) TargetName() string { return m.targetName }
func (m WhisperFindResultMap) MapId() uint32      { return m.mapId }
func (m WhisperFindResultMap) IncludeXY() bool    { return m.includeXY }
func (m WhisperFindResultMap) X() int16           { return m.x }
func (m WhisperFindResultMap) Y() int16           { return m.y }

func (m WhisperFindResultMap) Operation() string { return WhisperWriter }
func (m WhisperFindResultMap) String() string {
	return fmt.Sprintf("whisper find [%s] on map [%d]", m.targetName, m.mapId)
}

func (m WhisperFindResultMap) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.targetName)
		w.WriteByte(1)
		w.WriteInt(m.mapId)
		if m.includeXY {
			w.WriteShort(uint16(m.x))
			w.WriteShort(uint16(m.y))
		}
		return w.Bytes()
	}
}

func (m *WhisperFindResultMap) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.targetName = r.ReadAsciiString()
		_ = r.ReadByte() // findMode = 1
		m.mapId = r.ReadUint32()
		if m.includeXY {
			m.x = int16(r.ReadUint16())
			m.y = int16(r.ReadUint16())
		}
	}
}

// WhisperFindResultChannel - target is on a different channel
type WhisperFindResultChannel struct {
	mode       byte
	targetName string
	channelId  uint32
}

func NewWhisperFindResultChannel(mode byte, targetName string, channelId uint32) WhisperFindResultChannel {
	return WhisperFindResultChannel{mode: mode, targetName: targetName, channelId: channelId}
}

func (m WhisperFindResultChannel) Mode() byte        { return m.mode }
func (m WhisperFindResultChannel) TargetName() string { return m.targetName }
func (m WhisperFindResultChannel) ChannelId() uint32  { return m.channelId }

func (m WhisperFindResultChannel) Operation() string { return WhisperWriter }
func (m WhisperFindResultChannel) String() string {
	return fmt.Sprintf("whisper find [%s] on channel [%d]", m.targetName, m.channelId)
}

func (m WhisperFindResultChannel) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.targetName)
		w.WriteByte(3)
		w.WriteInt(m.channelId)
		return w.Bytes()
	}
}

func (m *WhisperFindResultChannel) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.targetName = r.ReadAsciiString()
		_ = r.ReadByte() // findMode = 3
		m.channelId = r.ReadUint32()
	}
}

// WhisperFindResultError - target not found
type WhisperFindResultError struct {
	mode       byte
	targetName string
}

func NewWhisperFindResultError(mode byte, targetName string) WhisperFindResultError {
	return WhisperFindResultError{mode: mode, targetName: targetName}
}

func (m WhisperFindResultError) Mode() byte        { return m.mode }
func (m WhisperFindResultError) TargetName() string { return m.targetName }

func (m WhisperFindResultError) Operation() string { return WhisperWriter }
func (m WhisperFindResultError) String() string {
	return fmt.Sprintf("whisper find error for [%s]", m.targetName)
}

func (m WhisperFindResultError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.targetName)
		w.WriteByte(0)
		w.WriteInt(0)
		return w.Bytes()
	}
}

func (m *WhisperFindResultError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.targetName = r.ReadAsciiString()
		_ = r.ReadByte()   // findMode = 0
		_ = r.ReadUint32() // 0
	}
}

// WhisperError - whisper blocked/disabled
type WhisperError struct {
	mode             byte
	targetName       string
	whispersEnabled  bool
}

func NewWhisperError(mode byte, targetName string, whispersEnabled bool) WhisperError {
	return WhisperError{mode: mode, targetName: targetName, whispersEnabled: whispersEnabled}
}

func (m WhisperError) Mode() byte             { return m.mode }
func (m WhisperError) TargetName() string      { return m.targetName }
func (m WhisperError) WhispersEnabled() bool   { return m.whispersEnabled }

func (m WhisperError) Operation() string { return WhisperWriter }
func (m WhisperError) String() string {
	return fmt.Sprintf("whisper error for [%s] enabled [%t]", m.targetName, m.whispersEnabled)
}

func (m WhisperError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.targetName)
		w.WriteBool(m.whispersEnabled)
		return w.Bytes()
	}
}

func (m *WhisperError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.targetName = r.ReadAsciiString()
		m.whispersEnabled = r.ReadBool()
	}
}

// WhisperWeather - GM weather message
type WhisperWeather struct {
	mode     byte
	fromName string
	message  string
}

func NewWhisperWeather(mode byte, fromName string, message string) WhisperWeather {
	return WhisperWeather{mode: mode, fromName: fromName, message: message}
}

func (m WhisperWeather) Mode() byte       { return m.mode }
func (m WhisperWeather) FromName() string  { return m.fromName }
func (m WhisperWeather) Message() string   { return m.message }

func (m WhisperWeather) Operation() string { return WhisperWriter }
func (m WhisperWeather) String() string {
	return fmt.Sprintf("whisper weather from [%s]", m.fromName)
}

func (m WhisperWeather) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.fromName)
		w.WriteBool(true)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *WhisperWeather) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.fromName = r.ReadAsciiString()
		_ = r.ReadBool() // always true
		m.message = r.ReadAsciiString()
	}
}
