package character

import (
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterStatusMessageWriter = "CharacterStatusMessage"

// StatusMessageDropPickUpItemUnavailable - item unavailable (-2)
type StatusMessageDropPickUpItemUnavailable struct {
	mode byte
}

func NewStatusMessageDropPickUpItemUnavailable(mode byte) StatusMessageDropPickUpItemUnavailable {
	return StatusMessageDropPickUpItemUnavailable{mode: mode}
}

func (m StatusMessageDropPickUpItemUnavailable) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageDropPickUpItemUnavailable) String() string {
	return fmt.Sprintf("drop pick up item unavailable, mode [%d]", m.mode)
}

func (m StatusMessageDropPickUpItemUnavailable) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt8(-2)
		return w.Bytes()
	}
}

func (m *StatusMessageDropPickUpItemUnavailable) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadInt8()
	}
}

// StatusMessageDropPickUpInventoryFull - inventory full (-3)
type StatusMessageDropPickUpInventoryFull struct {
	mode byte
}

func NewStatusMessageDropPickUpInventoryFull(mode byte) StatusMessageDropPickUpInventoryFull {
	return StatusMessageDropPickUpInventoryFull{mode: mode}
}

func (m StatusMessageDropPickUpInventoryFull) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageDropPickUpInventoryFull) String() string {
	return fmt.Sprintf("drop pick up inventory full, mode [%d]", m.mode)
}

func (m StatusMessageDropPickUpInventoryFull) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt8(-3)
		return w.Bytes()
	}
}

func (m *StatusMessageDropPickUpInventoryFull) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadInt8()
	}
}

// StatusMessageDropPickUpStackableItem - picked up stackable item
type StatusMessageDropPickUpStackableItem struct {
	mode   byte
	itemId uint32
	amount uint32
}

func NewStatusMessageDropPickUpStackableItem(mode byte, itemId uint32, amount uint32) StatusMessageDropPickUpStackableItem {
	return StatusMessageDropPickUpStackableItem{mode: mode, itemId: itemId, amount: amount}
}

func (m StatusMessageDropPickUpStackableItem) Operation() string {
	return CharacterStatusMessageWriter
}
func (m StatusMessageDropPickUpStackableItem) String() string {
	return fmt.Sprintf("drop pick up stackable item [%d] amount [%d]", m.itemId, m.amount)
}

func (m StatusMessageDropPickUpStackableItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt8(0)
		w.WriteInt(m.itemId)
		w.WriteInt(m.amount)
		return w.Bytes()
	}
}

func (m *StatusMessageDropPickUpStackableItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadInt8()
		m.itemId = r.ReadUint32()
		m.amount = r.ReadUint32()
	}
}

// StatusMessageDropPickUpUnStackableItem - picked up unstackable item
type StatusMessageDropPickUpUnStackableItem struct {
	mode   byte
	itemId uint32
}

func NewStatusMessageDropPickUpUnStackableItem(mode byte, itemId uint32) StatusMessageDropPickUpUnStackableItem {
	return StatusMessageDropPickUpUnStackableItem{mode: mode, itemId: itemId}
}

func (m StatusMessageDropPickUpUnStackableItem) Operation() string {
	return CharacterStatusMessageWriter
}
func (m StatusMessageDropPickUpUnStackableItem) String() string {
	return fmt.Sprintf("drop pick up unstackable item [%d]", m.itemId)
}

func (m StatusMessageDropPickUpUnStackableItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt8(2)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *StatusMessageDropPickUpUnStackableItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadInt8()
		m.itemId = r.ReadUint32()
	}
}

// StatusMessageDropPickUpMeso - picked up meso
type StatusMessageDropPickUpMeso struct {
	mode              byte
	partial           bool
	amount            uint32
	internetCafeBonus uint16
}

func NewStatusMessageDropPickUpMeso(mode byte, partial bool, amount uint32, internetCafeBonus uint16) StatusMessageDropPickUpMeso {
	return StatusMessageDropPickUpMeso{mode: mode, partial: partial, amount: amount, internetCafeBonus: internetCafeBonus}
}

func (m StatusMessageDropPickUpMeso) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageDropPickUpMeso) String() string {
	return fmt.Sprintf("drop pick up meso [%d] partial [%t]", m.amount, m.partial)
}

func (m StatusMessageDropPickUpMeso) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt8(1)
		w.WriteBool(m.partial)
		w.WriteInt(m.amount)
		w.WriteShort(m.internetCafeBonus)
		return w.Bytes()
	}
}

func (m *StatusMessageDropPickUpMeso) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadInt8()
		m.partial = r.ReadBool()
		m.amount = r.ReadUint32()
		m.internetCafeBonus = r.ReadUint16()
	}
}

// StatusMessageForfeitQuestRecord - quest forfeit
type StatusMessageForfeitQuestRecord struct {
	mode    byte
	questId uint16
}

func NewStatusMessageForfeitQuestRecord(mode byte, questId uint16) StatusMessageForfeitQuestRecord {
	return StatusMessageForfeitQuestRecord{mode: mode, questId: questId}
}

func (m StatusMessageForfeitQuestRecord) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageForfeitQuestRecord) String() string {
	return fmt.Sprintf("forfeit quest [%d]", m.questId)
}

func (m StatusMessageForfeitQuestRecord) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(m.questId)
		w.WriteByte(0)
		return w.Bytes()
	}
}

func (m *StatusMessageForfeitQuestRecord) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.questId = r.ReadUint16()
		_ = r.ReadByte()
	}
}

// StatusMessageUpdateQuestRecord - quest info update
type StatusMessageUpdateQuestRecord struct {
	mode    byte
	questId uint16
	info    string
}

func NewStatusMessageUpdateQuestRecord(mode byte, questId uint16, info string) StatusMessageUpdateQuestRecord {
	return StatusMessageUpdateQuestRecord{mode: mode, questId: questId, info: info}
}

func (m StatusMessageUpdateQuestRecord) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageUpdateQuestRecord) String() string {
	return fmt.Sprintf("update quest [%d] info [%s]", m.questId, m.info)
}

func (m StatusMessageUpdateQuestRecord) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(m.questId)
		w.WriteByte(1)
		w.WriteAsciiString(m.info)
		return w.Bytes()
	}
}

func (m *StatusMessageUpdateQuestRecord) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.questId = r.ReadUint16()
		_ = r.ReadByte()
		m.info = r.ReadAsciiString()
	}
}

// StatusMessageCompleteQuestRecord - quest completion
type StatusMessageCompleteQuestRecord struct {
	mode        byte
	questId     uint16
	completedAt time.Time
}

func NewStatusMessageCompleteQuestRecord(mode byte, questId uint16, completedAt time.Time) StatusMessageCompleteQuestRecord {
	return StatusMessageCompleteQuestRecord{mode: mode, questId: questId, completedAt: completedAt}
}

func (m StatusMessageCompleteQuestRecord) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageCompleteQuestRecord) String() string {
	return fmt.Sprintf("complete quest [%d]", m.questId)
}

func statusMessageMsTime(t time.Time) int64 {
	if t.IsZero() {
		return -1
	}
	return t.Unix()*int64(10000000) + int64(116444736000000000)
}

func statusMessageFromMsTime(v int64) time.Time {
	if v == -1 {
		return time.Time{}
	}
	unixSec := (v - int64(116444736000000000)) / int64(10000000)
	return time.Unix(unixSec, 0).UTC()
}

func (m StatusMessageCompleteQuestRecord) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(m.questId)
		w.WriteByte(2)
		w.WriteInt64(statusMessageMsTime(m.completedAt))
		return w.Bytes()
	}
}

func (m *StatusMessageCompleteQuestRecord) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.questId = r.ReadUint16()
		_ = r.ReadByte()
		m.completedAt = statusMessageFromMsTime(r.ReadInt64())
	}
}

// StatusMessageCashItemExpire - cash item expired
type StatusMessageCashItemExpire struct {
	mode   byte
	itemId uint32
}

func NewStatusMessageCashItemExpire(mode byte, itemId uint32) StatusMessageCashItemExpire {
	return StatusMessageCashItemExpire{mode: mode, itemId: itemId}
}

func (m StatusMessageCashItemExpire) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageCashItemExpire) String() string {
	return fmt.Sprintf("cash item expire [%d]", m.itemId)
}

func (m StatusMessageCashItemExpire) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *StatusMessageCashItemExpire) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itemId = r.ReadUint32()
	}
}

// StatusMessageIncreaseExperience - experience gain
type StatusMessageIncreaseExperience struct {
	mode                    byte
	white                   bool
	amount                  int32
	inChat                  bool
	monsterBookBonus        int32
	mobEventBonusPercentage byte
	partyBonusPercentage    byte
	weddingBonusEXP         int32
	playTimeHour            byte
	questBonusRate          byte
	questBonusRemainCount   byte
	partyBonusEventRate     byte
	partyBonusExp           int32
	itemBonusEXP            int32
	premiumIPExp            int32
	rainbowWeekEventEXP     int32
	partyEXPRingEXP         int32
	cakePieEventBonus       int32
}

func NewStatusMessageIncreaseExperience(mode byte, white bool, amount int32, inChat bool, monsterBookBonus int32,
	mobEventBonusPercentage byte, partyBonusPercentage byte, weddingBonusEXP int32, playTimeHour byte,
	questBonusRate byte, questBonusRemainCount byte, partyBonusEventRate byte, partyBonusExp int32,
	itemBonusEXP int32, premiumIPExp int32, rainbowWeekEventEXP int32, partyEXPRingEXP int32, cakePieEventBonus int32) StatusMessageIncreaseExperience {
	return StatusMessageIncreaseExperience{
		mode: mode, white: white, amount: amount, inChat: inChat, monsterBookBonus: monsterBookBonus,
		mobEventBonusPercentage: mobEventBonusPercentage, partyBonusPercentage: partyBonusPercentage,
		weddingBonusEXP: weddingBonusEXP, playTimeHour: playTimeHour, questBonusRate: questBonusRate,
		questBonusRemainCount: questBonusRemainCount, partyBonusEventRate: partyBonusEventRate,
		partyBonusExp: partyBonusExp, itemBonusEXP: itemBonusEXP, premiumIPExp: premiumIPExp,
		rainbowWeekEventEXP: rainbowWeekEventEXP, partyEXPRingEXP: partyEXPRingEXP, cakePieEventBonus: cakePieEventBonus,
	}
}

func (m StatusMessageIncreaseExperience) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageIncreaseExperience) String() string {
	return fmt.Sprintf("increase experience [%d] white [%t]", m.amount, m.white)
}

func (m StatusMessageIncreaseExperience) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(m.white)
		w.WriteInt32(m.amount)
		w.WriteBool(m.inChat)
		w.WriteInt32(m.monsterBookBonus)
		w.WriteByte(m.mobEventBonusPercentage)
		w.WriteByte(m.partyBonusPercentage)
		w.WriteInt32(m.weddingBonusEXP)
		if m.mobEventBonusPercentage > 0 {
			w.WriteByte(m.playTimeHour)
		}
		if m.inChat {
			w.WriteByte(m.questBonusRate)
			if m.questBonusRate > 0 {
				w.WriteByte(m.questBonusRemainCount)
			}
		}
		w.WriteByte(m.partyBonusEventRate)
		w.WriteInt32(m.partyBonusExp)
		w.WriteInt32(m.itemBonusEXP)
		w.WriteInt32(m.premiumIPExp)
		w.WriteInt32(m.rainbowWeekEventEXP)
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			w.WriteInt32(m.partyEXPRingEXP)
			w.WriteInt32(m.cakePieEventBonus)
		}
		return w.Bytes()
	}
}

func (m *StatusMessageIncreaseExperience) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)
		m.mode = r.ReadByte()
		m.white = r.ReadBool()
		m.amount = r.ReadInt32()
		m.inChat = r.ReadBool()
		m.monsterBookBonus = r.ReadInt32()
		m.mobEventBonusPercentage = r.ReadByte()
		m.partyBonusPercentage = r.ReadByte()
		m.weddingBonusEXP = r.ReadInt32()
		if m.mobEventBonusPercentage > 0 {
			m.playTimeHour = r.ReadByte()
		}
		if m.inChat {
			m.questBonusRate = r.ReadByte()
			if m.questBonusRate > 0 {
				m.questBonusRemainCount = r.ReadByte()
			}
		}
		m.partyBonusEventRate = r.ReadByte()
		m.partyBonusExp = r.ReadInt32()
		m.itemBonusEXP = r.ReadInt32()
		m.premiumIPExp = r.ReadInt32()
		m.rainbowWeekEventEXP = r.ReadInt32()
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.partyEXPRingEXP = r.ReadInt32()
			m.cakePieEventBonus = r.ReadInt32()
		}
	}
}

// StatusMessageIncreaseSkillPoint - SP gain
type StatusMessageIncreaseSkillPoint struct {
	mode   byte
	jobId  uint16
	amount byte
}

func NewStatusMessageIncreaseSkillPoint(mode byte, jobId uint16, amount byte) StatusMessageIncreaseSkillPoint {
	return StatusMessageIncreaseSkillPoint{mode: mode, jobId: jobId, amount: amount}
}

func (m StatusMessageIncreaseSkillPoint) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageIncreaseSkillPoint) String() string {
	return fmt.Sprintf("increase skill point job [%d] amount [%d]", m.jobId, m.amount)
}

func (m StatusMessageIncreaseSkillPoint) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(m.jobId)
		w.WriteByte(m.amount)
		return w.Bytes()
	}
}

func (m *StatusMessageIncreaseSkillPoint) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.jobId = r.ReadUint16()
		m.amount = r.ReadByte()
	}
}

// StatusMessageIncreaseFame - fame change
type StatusMessageIncreaseFame struct {
	mode   byte
	amount int32
}

func NewStatusMessageIncreaseFame(mode byte, amount int32) StatusMessageIncreaseFame {
	return StatusMessageIncreaseFame{mode: mode, amount: amount}
}

func (m StatusMessageIncreaseFame) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageIncreaseFame) String() string {
	return fmt.Sprintf("increase fame [%d]", m.amount)
}

func (m StatusMessageIncreaseFame) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt32(m.amount)
		return w.Bytes()
	}
}

func (m *StatusMessageIncreaseFame) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.amount = r.ReadInt32()
	}
}

// StatusMessageIncreaseMeso - meso change
type StatusMessageIncreaseMeso struct {
	mode   byte
	amount int32
}

func NewStatusMessageIncreaseMeso(mode byte, amount int32) StatusMessageIncreaseMeso {
	return StatusMessageIncreaseMeso{mode: mode, amount: amount}
}

func (m StatusMessageIncreaseMeso) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageIncreaseMeso) String() string {
	return fmt.Sprintf("increase meso [%d]", m.amount)
}

func (m StatusMessageIncreaseMeso) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt32(m.amount)
		return w.Bytes()
	}
}

func (m *StatusMessageIncreaseMeso) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.amount = r.ReadInt32()
	}
}

// StatusMessageIncreaseGuildPoint - guild point change
type StatusMessageIncreaseGuildPoint struct {
	mode   byte
	amount int32
}

func NewStatusMessageIncreaseGuildPoint(mode byte, amount int32) StatusMessageIncreaseGuildPoint {
	return StatusMessageIncreaseGuildPoint{mode: mode, amount: amount}
}

func (m StatusMessageIncreaseGuildPoint) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageIncreaseGuildPoint) String() string {
	return fmt.Sprintf("increase guild point [%d]", m.amount)
}

func (m StatusMessageIncreaseGuildPoint) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt32(m.amount)
		return w.Bytes()
	}
}

func (m *StatusMessageIncreaseGuildPoint) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.amount = r.ReadInt32()
	}
}

// StatusMessageGiveBuff - buff given
type StatusMessageGiveBuff struct {
	mode   byte
	itemId uint32
}

func NewStatusMessageGiveBuff(mode byte, itemId uint32) StatusMessageGiveBuff {
	return StatusMessageGiveBuff{mode: mode, itemId: itemId}
}

func (m StatusMessageGiveBuff) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageGiveBuff) String() string {
	return fmt.Sprintf("give buff item [%d]", m.itemId)
}

func (m StatusMessageGiveBuff) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *StatusMessageGiveBuff) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itemId = r.ReadUint32()
	}
}

// StatusMessageGeneralItemExpire - general items expired
type StatusMessageGeneralItemExpire struct {
	mode    byte
	itemIds []uint32
}

func NewStatusMessageGeneralItemExpire(mode byte, itemIds []uint32) StatusMessageGeneralItemExpire {
	return StatusMessageGeneralItemExpire{mode: mode, itemIds: itemIds}
}

func (m StatusMessageGeneralItemExpire) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageGeneralItemExpire) String() string {
	return fmt.Sprintf("general item expire count [%d]", len(m.itemIds))
}

func (m StatusMessageGeneralItemExpire) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(byte(len(m.itemIds)))
		for _, itemId := range m.itemIds {
			w.WriteInt(itemId)
		}
		return w.Bytes()
	}
}

func (m *StatusMessageGeneralItemExpire) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := r.ReadByte()
		m.itemIds = make([]uint32, count)
		for i := byte(0); i < count; i++ {
			m.itemIds[i] = r.ReadUint32()
		}
	}
}

// StatusMessageSystemMessage - system message in chat
type StatusMessageSystemMessage struct {
	mode    byte
	message string
}

func NewStatusMessageSystemMessage(mode byte, message string) StatusMessageSystemMessage {
	return StatusMessageSystemMessage{mode: mode, message: message}
}

func (m StatusMessageSystemMessage) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageSystemMessage) String() string {
	return fmt.Sprintf("system message [%s]", m.message)
}

func (m StatusMessageSystemMessage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *StatusMessageSystemMessage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
	}
}

// StatusMessageQuestRecordEx - quest record ex
type StatusMessageQuestRecordEx struct {
	mode    byte
	questId uint16
	info    string
}

func NewStatusMessageQuestRecordEx(mode byte, questId uint16, info string) StatusMessageQuestRecordEx {
	return StatusMessageQuestRecordEx{mode: mode, questId: questId, info: info}
}

func (m StatusMessageQuestRecordEx) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageQuestRecordEx) String() string {
	return fmt.Sprintf("quest record ex [%d] info [%s]", m.questId, m.info)
}

func (m StatusMessageQuestRecordEx) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(m.questId)
		w.WriteAsciiString(m.info)
		return w.Bytes()
	}
}

func (m *StatusMessageQuestRecordEx) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.questId = r.ReadUint16()
		m.info = r.ReadAsciiString()
	}
}

// StatusMessageItemProtectExpire - item protection expired
type StatusMessageItemProtectExpire struct {
	mode    byte
	itemIds []uint32
}

func NewStatusMessageItemProtectExpire(mode byte, itemIds []uint32) StatusMessageItemProtectExpire {
	return StatusMessageItemProtectExpire{mode: mode, itemIds: itemIds}
}

func (m StatusMessageItemProtectExpire) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageItemProtectExpire) String() string {
	return fmt.Sprintf("item protect expire count [%d]", len(m.itemIds))
}

func (m StatusMessageItemProtectExpire) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(byte(len(m.itemIds)))
		for _, itemId := range m.itemIds {
			w.WriteInt(itemId)
		}
		return w.Bytes()
	}
}

func (m *StatusMessageItemProtectExpire) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := r.ReadByte()
		m.itemIds = make([]uint32, count)
		for i := byte(0); i < count; i++ {
			m.itemIds[i] = r.ReadUint32()
		}
	}
}

// StatusMessageItemExpireReplace - item expire replace messages
type StatusMessageItemExpireReplace struct {
	mode     byte
	messages []string
}

func NewStatusMessageItemExpireReplace(mode byte, messages []string) StatusMessageItemExpireReplace {
	return StatusMessageItemExpireReplace{mode: mode, messages: messages}
}

func (m StatusMessageItemExpireReplace) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageItemExpireReplace) String() string {
	return fmt.Sprintf("item expire replace count [%d]", len(m.messages))
}

func (m StatusMessageItemExpireReplace) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(byte(len(m.messages)))
		for _, message := range m.messages {
			w.WriteAsciiString(message)
		}
		return w.Bytes()
	}
}

func (m *StatusMessageItemExpireReplace) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := r.ReadByte()
		m.messages = make([]string, count)
		for i := byte(0); i < count; i++ {
			m.messages[i] = r.ReadAsciiString()
		}
	}
}

// StatusMessageSkillExpire - skills expired
type StatusMessageSkillExpire struct {
	mode     byte
	skillIds []uint32
}

func NewStatusMessageSkillExpire(mode byte, skillIds []uint32) StatusMessageSkillExpire {
	return StatusMessageSkillExpire{mode: mode, skillIds: skillIds}
}

func (m StatusMessageSkillExpire) Operation() string { return CharacterStatusMessageWriter }
func (m StatusMessageSkillExpire) String() string {
	return fmt.Sprintf("skill expire count [%d]", len(m.skillIds))
}

func (m StatusMessageSkillExpire) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(byte(len(m.skillIds)))
		for _, skillId := range m.skillIds {
			w.WriteInt(skillId)
		}
		return w.Bytes()
	}
}

func (m *StatusMessageSkillExpire) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := r.ReadByte()
		m.skillIds = make([]uint32, count)
		for i := byte(0); i < count; i++ {
			m.skillIds[i] = r.ReadUint32()
		}
	}
}
