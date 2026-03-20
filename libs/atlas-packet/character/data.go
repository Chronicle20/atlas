package character

import (
	"context"

	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type CharacterStats struct {
	Id        uint32
	Name      string // max 13 chars, padded with zeros
	Gender    byte
	SkinColor byte
	Face      uint32
	Hair      uint32
	PetIds    [3]uint64
	Level     byte
	JobId     uint16
	Str       uint16
	Dex       uint16
	Int       uint16
	Luk       uint16
	Hp        uint16
	MaxHp     uint16
	Mp        uint16
	MaxMp     uint16
	Ap        uint16
	Sp        uint16
	Exp       uint32
	Fame      int16
	GachaExp  uint32
	MapId     uint32
	SpawnPoint byte
}

type InventoryData struct {
	EquipCapacity byte
	UseCapacity   byte
	SetupCapacity byte
	EtcCapacity   byte
	CashCapacity  byte
	Timestamp     int64
	RegularEquip  []model.Asset
	CashEquip     []model.Asset
	EquipInv      []model.Asset
	UseInv        []model.Asset
	SetupInv      []model.Asset
	EtcInv        []model.Asset
	CashInv       []model.Asset
}

type SkillEntry struct {
	Id         uint32
	Level      uint32
	Expiration int64
	MasterLevel uint32
	FourthJob  bool
}

type CooldownEntry struct {
	SkillId   uint32
	Remaining uint16
}

type QuestProgress struct {
	QuestId  uint16
	Progress string
}

type QuestCompleted struct {
	QuestId     uint16
	CompletedAt int64
}

type CharacterData struct {
	Stats          CharacterStats
	BuddyCapacity  byte
	Meso           uint32
	Inventory      InventoryData
	Skills         []SkillEntry
	Cooldowns      []CooldownEntry
	StartedQuests  []QuestProgress
	CompletedQuests []QuestCompleted
}

func (m CharacterData) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		// dbcharFlag
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			w.WriteInt64(-1)
			w.WriteByte(0) // SN list size
		} else {
			w.WriteInt16(-1)
		}

		m.encodeStats(w, t)
		w.WriteByte(m.BuddyCapacity)

		// linked name
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			w.WriteByte(0) // not linked
		}
		w.WriteInt(m.Meso)

		// JMS extra
		if t.Region() == "JMS" {
			w.WriteInt(m.Stats.Id)
			w.WriteInt(0) // dama
			w.WriteInt(0)
		}

		m.encodeInventory(l, ctx, options, w)
		m.encodeSkills(w, t)
		m.encodeQuests(w, t)
		m.encodeMiniGame(w)
		m.encodeRings(w, t)
		m.encodeTeleports(w, t)
		if t.Region() == "JMS" {
			w.WriteShort(0)
		}

		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			m.encodeMonsterBook(w)
			if t.Region() == "GMS" {
				m.encodeNewYear(w)
				m.encodeArea(w)
			} else if t.Region() == "JMS" {
				w.WriteShort(0)
			}
			w.WriteShort(0)
		}
		return w.Bytes()
	}
}

func (m *CharacterData) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)

		// dbcharFlag
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			_ = r.ReadInt64()
			_ = r.ReadByte() // SN list size
		} else {
			_ = r.ReadInt16()
		}

		m.decodeStats(r, t)
		m.BuddyCapacity = r.ReadByte()

		// linked name
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			linked := r.ReadByte()
			if linked == 1 {
				_ = r.ReadAsciiString()
			}
		}
		m.Meso = r.ReadUint32()

		// JMS extra
		if t.Region() == "JMS" {
			_ = r.ReadUint32() // characterId
			_ = r.ReadUint32() // dama
			_ = r.ReadUint32()
		}

		m.decodeInventory(l, ctx, r, options)
		m.decodeSkills(r, t)
		m.decodeQuests(r, t)
		m.decodeMiniGame(r)
		m.decodeRings(r, t)
		m.decodeTeleports(r, t)
		if t.Region() == "JMS" {
			_ = r.ReadUint16()
		}

		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			m.decodeMonsterBook(r)
			if t.Region() == "GMS" {
				m.decodeNewYear(r)
				m.decodeArea(r)
			} else if t.Region() == "JMS" {
				_ = r.ReadUint16()
			}
			_ = r.ReadUint16()
		}
	}
}

// Stats encoding/decoding

func (m *CharacterData) encodeStats(w *response.Writer, t tenant.Model) {
	w.WriteInt(m.Stats.Id)

	name := m.Stats.Name
	if len(name) > 13 {
		name = name[:13]
	}
	padSize := 13 - len(name)
	w.WriteByteArray([]byte(name))
	for i := 0; i < padSize; i++ {
		w.WriteByte(0)
	}

	w.WriteByte(m.Stats.Gender)
	w.WriteByte(m.Stats.SkinColor)
	w.WriteInt(m.Stats.Face)
	w.WriteInt(m.Stats.Hair)

	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		for i := 0; i < 3; i++ {
			w.WriteLong(m.Stats.PetIds[i])
		}
	} else {
		w.WriteLong(m.Stats.PetIds[0])
	}

	w.WriteByte(m.Stats.Level)
	w.WriteShort(m.Stats.JobId)
	w.WriteShort(m.Stats.Str)
	w.WriteShort(m.Stats.Dex)
	w.WriteShort(m.Stats.Int)
	w.WriteShort(m.Stats.Luk)
	w.WriteShort(m.Stats.Hp)
	w.WriteShort(m.Stats.MaxHp)
	w.WriteShort(m.Stats.Mp)
	w.WriteShort(m.Stats.MaxMp)
	w.WriteShort(m.Stats.Ap)
	w.WriteShort(m.Stats.Sp)
	w.WriteInt(m.Stats.Exp)
	w.WriteInt16(m.Stats.Fame)

	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		w.WriteInt(m.Stats.GachaExp)
	}
	w.WriteInt(m.Stats.MapId)
	w.WriteByte(m.Stats.SpawnPoint)

	if t.Region() == "GMS" {
		if t.MajorVersion() > 12 {
			w.WriteInt(0)
		} else {
			w.WriteInt64(0)
			w.WriteInt(0)
			w.WriteInt(0)
		}
		if t.MajorVersion() >= 87 {
			w.WriteShort(0) // nSubJob
		}
	} else if t.Region() == "JMS" {
		w.WriteShort(0)
		w.WriteLong(0)
		w.WriteInt(0)
		w.WriteInt(0)
		w.WriteInt(0)
	}
}

func (m *CharacterData) decodeStats(r *request.Reader, t tenant.Model) {
	m.Stats.Id = r.ReadUint32()

	nameBytes := r.ReadBytes(13)
	end := 13
	for i, b := range nameBytes {
		if b == 0 {
			end = i
			break
		}
	}
	m.Stats.Name = string(nameBytes[:end])

	m.Stats.Gender = r.ReadByte()
	m.Stats.SkinColor = r.ReadByte()
	m.Stats.Face = r.ReadUint32()
	m.Stats.Hair = r.ReadUint32()

	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		for i := 0; i < 3; i++ {
			m.Stats.PetIds[i] = r.ReadUint64()
		}
	} else {
		m.Stats.PetIds[0] = r.ReadUint64()
	}

	m.Stats.Level = r.ReadByte()
	m.Stats.JobId = r.ReadUint16()
	m.Stats.Str = r.ReadUint16()
	m.Stats.Dex = r.ReadUint16()
	m.Stats.Int = r.ReadUint16()
	m.Stats.Luk = r.ReadUint16()
	m.Stats.Hp = r.ReadUint16()
	m.Stats.MaxHp = r.ReadUint16()
	m.Stats.Mp = r.ReadUint16()
	m.Stats.MaxMp = r.ReadUint16()
	m.Stats.Ap = r.ReadUint16()
	m.Stats.Sp = r.ReadUint16()
	m.Stats.Exp = r.ReadUint32()
	m.Stats.Fame = r.ReadInt16()

	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		m.Stats.GachaExp = r.ReadUint32()
	}
	m.Stats.MapId = r.ReadUint32()
	m.Stats.SpawnPoint = r.ReadByte()

	if t.Region() == "GMS" {
		if t.MajorVersion() > 12 {
			_ = r.ReadUint32()
		} else {
			_ = r.ReadInt64()
			_ = r.ReadUint32()
			_ = r.ReadUint32()
		}
		if t.MajorVersion() >= 87 {
			_ = r.ReadUint16() // nSubJob
		}
	} else if t.Region() == "JMS" {
		_ = r.ReadUint16()
		_ = r.ReadUint64()
		_ = r.ReadUint32()
		_ = r.ReadUint32()
		_ = r.ReadUint32()
	}
}

// Inventory encoding/decoding

func (m *CharacterData) encodeInventory(l logrus.FieldLogger, ctx context.Context, options map[string]interface{}, w *response.Writer) {
	t := tenant.MustFromContext(ctx)

	if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
		w.WriteByte(m.Inventory.EquipCapacity)
		w.WriteByte(m.Inventory.UseCapacity)
		w.WriteByte(m.Inventory.SetupCapacity)
		w.WriteByte(m.Inventory.EtcCapacity)
		w.WriteByte(m.Inventory.CashCapacity)
	}

	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		w.WriteInt64(m.Inventory.Timestamp)
	}

	// Regular equipment
	for i := range m.Inventory.RegularEquip {
		w.WriteByteArray(m.Inventory.RegularEquip[i].Encode(l, ctx)(options))
	}
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		w.WriteShort(0)
	} else {
		w.WriteByte(0)
	}

	// Cash equipment
	for i := range m.Inventory.CashEquip {
		w.WriteByteArray(m.Inventory.CashEquip[i].Encode(l, ctx)(options))
	}
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		w.WriteShort(0)
	} else {
		w.WriteByte(0)
	}

	// Equipable inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		w.WriteByte(m.Inventory.EquipCapacity)
	}
	for i := range m.Inventory.EquipInv {
		w.WriteByteArray(m.Inventory.EquipInv[i].Encode(l, ctx)(options))
	}
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		w.WriteInt(0)
	} else {
		w.WriteByte(0)
	}

	// Use inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		w.WriteByte(m.Inventory.UseCapacity)
	}
	for i := range m.Inventory.UseInv {
		w.WriteByteArray(m.Inventory.UseInv[i].Encode(l, ctx)(options))
	}
	w.WriteByte(0)

	// Setup inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		w.WriteByte(m.Inventory.SetupCapacity)
	}
	for i := range m.Inventory.SetupInv {
		w.WriteByteArray(m.Inventory.SetupInv[i].Encode(l, ctx)(options))
	}
	w.WriteByte(0)

	// Etc inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		w.WriteByte(m.Inventory.EtcCapacity)
	}
	for i := range m.Inventory.EtcInv {
		w.WriteByteArray(m.Inventory.EtcInv[i].Encode(l, ctx)(options))
	}
	w.WriteByte(0)

	// Cash inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		w.WriteByte(m.Inventory.CashCapacity)
	}
	for i := range m.Inventory.CashInv {
		w.WriteByteArray(m.Inventory.CashInv[i].Encode(l, ctx)(options))
	}
	w.WriteByte(0)
}

func (m *CharacterData) decodeInventory(l logrus.FieldLogger, ctx context.Context, r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)

	if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
		m.Inventory.EquipCapacity = r.ReadByte()
		m.Inventory.UseCapacity = r.ReadByte()
		m.Inventory.SetupCapacity = r.ReadByte()
		m.Inventory.EtcCapacity = r.ReadByte()
		m.Inventory.CashCapacity = r.ReadByte()
	}

	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		m.Inventory.Timestamp = r.ReadInt64()
	}

	// Regular equipment: slot is negative (equipped items)
	m.Inventory.RegularEquip = decodeEquipmentSection(l, ctx, r, options, t, false)

	// Cash equipment: slot is negative with -100 offset
	m.Inventory.CashEquip = decodeEquipmentSection(l, ctx, r, options, t, true)

	// Equipable inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		m.Inventory.EquipCapacity = r.ReadByte()
	}
	m.Inventory.EquipInv = decodeEquipableInventorySection(l, ctx, r, options, t)

	// Use inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		m.Inventory.UseCapacity = r.ReadByte()
	}
	m.Inventory.UseInv = decodeStackableSection(l, ctx, r, options)

	// Setup inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		m.Inventory.SetupCapacity = r.ReadByte()
	}
	m.Inventory.SetupInv = decodeStackableSection(l, ctx, r, options)

	// Etc inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		m.Inventory.EtcCapacity = r.ReadByte()
	}
	m.Inventory.EtcInv = decodeStackableSection(l, ctx, r, options)

	// Cash inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		m.Inventory.CashCapacity = r.ReadByte()
	}
	m.Inventory.CashInv = decodeStackableSection(l, ctx, r, options)
}

func decodeEquipmentSection(l logrus.FieldLogger, ctx context.Context, r *request.Reader, options map[string]interface{}, t tenant.Model, cash bool) []model.Asset {
	var assets []model.Asset
	for {
		var wireSlot uint16
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			wireSlot = r.ReadUint16()
		} else {
			wireSlot = uint16(r.ReadByte())
		}
		if wireSlot == 0 {
			break
		}
		var slot int16
		if cash {
			slot = -int16(wireSlot) - 100
		} else {
			slot = -int16(wireSlot)
		}
		a := model.NewAsset(false, slot, 0, model.FromMsTime(0))
		a.Decode(l, ctx)(r, options)
		assets = append(assets, a)
	}
	return assets
}

func decodeEquipableInventorySection(l logrus.FieldLogger, ctx context.Context, r *request.Reader, options map[string]interface{}, t tenant.Model) []model.Asset {
	var assets []model.Asset
	for {
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			wireSlot := r.ReadUint16()
			if wireSlot == 0 {
				_ = r.ReadUint16() // consume remaining 2 bytes of WriteInt(0) terminator
				break
			}
			a := model.NewAsset(false, int16(wireSlot), 0, model.FromMsTime(0))
			a.Decode(l, ctx)(r, options)
			assets = append(assets, a)
		} else {
			wireSlot := r.ReadByte()
			if wireSlot == 0 {
				break
			}
			a := model.NewAsset(false, int16(wireSlot), 0, model.FromMsTime(0))
			a.Decode(l, ctx)(r, options)
			assets = append(assets, a)
		}
	}
	return assets
}

func decodeStackableSection(l logrus.FieldLogger, ctx context.Context, r *request.Reader, options map[string]interface{}) []model.Asset {
	var assets []model.Asset
	for {
		wireSlot := r.ReadInt8()
		if wireSlot == 0 {
			break
		}
		a := model.NewAsset(false, int16(wireSlot), 0, model.FromMsTime(0))
		a.Decode(l, ctx)(r, options)
		assets = append(assets, a)
	}
	return assets
}

// Skills encoding/decoding

func (m *CharacterData) encodeSkills(w *response.Writer, t tenant.Model) {
	w.WriteShort(uint16(len(m.Skills)))
	for _, s := range m.Skills {
		w.WriteInt(s.Id)
		w.WriteInt(s.Level)
		w.WriteInt64(s.Expiration)
		if s.FourthJob {
			w.WriteInt(s.MasterLevel)
		}
	}

	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		w.WriteShort(uint16(len(m.Cooldowns)))
		for _, cd := range m.Cooldowns {
			w.WriteInt(cd.SkillId)
			w.WriteShort(cd.Remaining)
		}
	}
}

func (m *CharacterData) decodeSkills(r *request.Reader, t tenant.Model) {
	skillCount := r.ReadUint16()
	m.Skills = make([]SkillEntry, skillCount)
	for i := uint16(0); i < skillCount; i++ {
		m.Skills[i].Id = r.ReadUint32()
		m.Skills[i].Level = r.ReadUint32()
		m.Skills[i].Expiration = r.ReadInt64()
		jobId := job.IdFromSkillId(skill.Id(m.Skills[i].Id))
		m.Skills[i].FourthJob = job.IsFourthJob(jobId)
		if m.Skills[i].FourthJob {
			m.Skills[i].MasterLevel = r.ReadUint32()
		}
	}

	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		cdCount := r.ReadUint16()
		m.Cooldowns = make([]CooldownEntry, cdCount)
		for i := uint16(0); i < cdCount; i++ {
			m.Cooldowns[i].SkillId = r.ReadUint32()
			m.Cooldowns[i].Remaining = r.ReadUint16()
		}
	}
}

// Quest encoding/decoding

func (m *CharacterData) encodeQuests(w *response.Writer, t tenant.Model) {
	w.WriteShort(uint16(len(m.StartedQuests)))
	for _, q := range m.StartedQuests {
		w.WriteShort(q.QuestId)
		w.WriteAsciiString(q.Progress)
	}

	if t.Region() == "JMS" {
		w.WriteShort(0)
	}

	if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
		w.WriteShort(uint16(len(m.CompletedQuests)))
		for _, q := range m.CompletedQuests {
			w.WriteShort(q.QuestId)
			w.WriteInt64(q.CompletedAt)
		}
	}
}

func (m *CharacterData) decodeQuests(r *request.Reader, t tenant.Model) {
	startedCount := r.ReadUint16()
	m.StartedQuests = make([]QuestProgress, startedCount)
	for i := uint16(0); i < startedCount; i++ {
		m.StartedQuests[i].QuestId = r.ReadUint16()
		m.StartedQuests[i].Progress = r.ReadAsciiString()
	}

	if t.Region() == "JMS" {
		_ = r.ReadUint16()
	}

	if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
		completedCount := r.ReadUint16()
		m.CompletedQuests = make([]QuestCompleted, completedCount)
		for i := uint16(0); i < completedCount; i++ {
			m.CompletedQuests[i].QuestId = r.ReadUint16()
			m.CompletedQuests[i].CompletedAt = r.ReadInt64()
		}
	}
}

// Zero-value sections — these always write fixed zero patterns

func (m *CharacterData) encodeMiniGame(w *response.Writer) {
	w.WriteShort(0)
}

func (m *CharacterData) decodeMiniGame(r *request.Reader) {
	_ = r.ReadUint16()
}

func (m *CharacterData) encodeRings(w *response.Writer, t tenant.Model) {
	w.WriteShort(0) // crush rings
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		w.WriteShort(0) // friendship rings
		w.WriteShort(0) // partner
	}
}

func (m *CharacterData) decodeRings(r *request.Reader, t tenant.Model) {
	_ = r.ReadUint16() // crush rings
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		_ = r.ReadUint16() // friendship rings
		_ = r.ReadUint16() // partner
	}
}

func (m *CharacterData) encodeTeleports(w *response.Writer, t tenant.Model) {
	for i := 0; i < 5; i++ {
		w.WriteInt(uint32(_map.EmptyMapId))
	}
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		for i := 0; i < 10; i++ {
			w.WriteInt(uint32(_map.EmptyMapId))
		}
	}
}

func (m *CharacterData) decodeTeleports(r *request.Reader, t tenant.Model) {
	for i := 0; i < 5; i++ {
		_ = r.ReadUint32()
	}
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		for i := 0; i < 10; i++ {
			_ = r.ReadUint32()
		}
	}
}

func (m *CharacterData) encodeMonsterBook(w *response.Writer) {
	w.WriteInt(0)   // cover id
	w.WriteByte(0)
	w.WriteShort(0) // card count
}

func (m *CharacterData) decodeMonsterBook(r *request.Reader) {
	_ = r.ReadUint32() // cover id
	_ = r.ReadByte()
	_ = r.ReadUint16() // card count
}

func (m *CharacterData) encodeNewYear(w *response.Writer) {
	w.WriteShort(0)
}

func (m *CharacterData) decodeNewYear(r *request.Reader) {
	_ = r.ReadUint16()
}

func (m *CharacterData) encodeArea(w *response.Writer) {
	w.WriteShort(0)
}

func (m *CharacterData) decodeArea(r *request.Reader) {
	_ = r.ReadUint16()
}
