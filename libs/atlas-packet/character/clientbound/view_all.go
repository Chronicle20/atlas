package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterViewAllWriter = "CharacterViewAll"

// CharacterViewAllCount — response with world count
type CharacterViewAllCount struct {
	code       byte
	worldCount uint32
	unk        uint32
}

func NewCharacterViewAllCount(code byte, worldCount uint32, unk uint32) CharacterViewAllCount {
	return CharacterViewAllCount{code: code, worldCount: worldCount, unk: unk}
}

func (m CharacterViewAllCount) Code() byte        { return m.code }
func (m CharacterViewAllCount) WorldCount() uint32 { return m.worldCount }
func (m CharacterViewAllCount) Unk() uint32        { return m.unk }
func (m CharacterViewAllCount) Operation() string  { return CharacterViewAllWriter }
func (m CharacterViewAllCount) String() string {
	return fmt.Sprintf("code [%d], worldCount [%d]", m.code, m.worldCount)
}

func (m CharacterViewAllCount) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.code)
		w.WriteInt(m.worldCount)
		w.WriteInt(m.unk)
		return w.Bytes()
	}
}

func (m *CharacterViewAllCount) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.code = r.ReadByte()
		m.worldCount = r.ReadUint32()
		m.unk = r.ReadUint32()
	}
}

// CharacterViewAllCharacters — response with characters for a world
type CharacterViewAllCharacters struct {
	code       byte
	worldId    world.Id
	characters []model.CharacterListEntry
}

func NewCharacterViewAllCharacters(code byte, worldId world.Id, characters []model.CharacterListEntry) CharacterViewAllCharacters {
	return CharacterViewAllCharacters{code: code, worldId: worldId, characters: characters}
}

func (m CharacterViewAllCharacters) Code() byte                              { return m.code }
func (m CharacterViewAllCharacters) WorldId() world.Id                       { return m.worldId }
func (m CharacterViewAllCharacters) Characters() []model.CharacterListEntry  { return m.characters }
func (m CharacterViewAllCharacters) Operation() string                       { return CharacterViewAllWriter }
func (m CharacterViewAllCharacters) String() string {
	return fmt.Sprintf("code [%d], worldId [%d], characters [%d]", m.code, m.worldId, len(m.characters))
}

func (m CharacterViewAllCharacters) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.code)
		w.WriteByte(byte(m.worldId))
		w.WriteByte(byte(len(m.characters)))
		for _, c := range m.characters {
			c.Write(l, ctx, w, options, true)
		}
		if t.Region() == "GMS" && t.MajorVersion() > 87 {
			w.WriteByte(1) // PIC handling
		}
		return w.Bytes()
	}
}

func (m *CharacterViewAllCharacters) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.code = r.ReadByte()
		m.worldId = world.Id(r.ReadByte())
		count := r.ReadByte()
		m.characters = make([]model.CharacterListEntry, count)
		for i := byte(0); i < count; i++ {
			m.characters[i].Read(l, ctx, r, options, true)
		}
		if t.Region() == "GMS" && t.MajorVersion() > 87 {
			_ = r.ReadByte() // PIC handling
		}
	}
}

// CharacterViewAllSearchFailed — simple error response
type CharacterViewAllSearchFailed struct {
	code byte
}

func NewCharacterViewAllSearchFailed(code byte) CharacterViewAllSearchFailed {
	return CharacterViewAllSearchFailed{code: code}
}

func (m CharacterViewAllSearchFailed) Code() byte        { return m.code }
func (m CharacterViewAllSearchFailed) Operation() string  { return CharacterViewAllWriter }
func (m CharacterViewAllSearchFailed) String() string     { return fmt.Sprintf("searchFailed code [%d]", m.code) }

func (m CharacterViewAllSearchFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.code)
		return w.Bytes()
	}
}

func (m *CharacterViewAllSearchFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.code = r.ReadByte()
	}
}

// CharacterViewAllError — simple error response
type CharacterViewAllError struct {
	code byte
}

func NewCharacterViewAllError(code byte) CharacterViewAllError {
	return CharacterViewAllError{code: code}
}

func (m CharacterViewAllError) Code() byte        { return m.code }
func (m CharacterViewAllError) Operation() string  { return CharacterViewAllWriter }
func (m CharacterViewAllError) String() string     { return fmt.Sprintf("error code [%d]", m.code) }

func (m CharacterViewAllError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.code)
		return w.Bytes()
	}
}

func (m *CharacterViewAllError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.code = r.ReadByte()
	}
}
