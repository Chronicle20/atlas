package model

import (
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ()

type Movement struct {
	StartX   int16
	StartY   int16
	Elements []Element
}

func (m *Movement) Encode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(m.StartX)
		w.WriteInt16(m.StartY)
		w.WriteByte(byte(len(m.Elements)))
		for _, element := range m.Elements {
			element.Encode(l, tenant, options)(w)
		}
	}
}

func (m *Movement) Decode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		m.StartX = r.ReadInt16()
		m.StartY = r.ReadInt16()

		numElems := r.ReadByte()
		var elems = make([]Element, numElems)
		for i := byte(0); i < numElems; i++ {
			elem := Element{}
			elem.StartX = m.StartX
			elem.StartY = m.StartY
			elem.Decode(l, tenant, options)(r)
			elems[i] = elem
		}
		m.Elements = elems
	}
}

type Element struct {
	StartX      int16
	StartY      int16
	BMoveAction byte
	BStat       byte
	X           int16
	Y           int16
	Vx          int16
	Vy          int16
	Fh          int16
	FhFallStart int16
	XOffset     int16
	YOffset     int16
	TElapse     int16
	ElemType    byte
}

func (m *Element) Encode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		var elemType = m.ElemType
		w.WriteByte(m.ElemType)
		if isMovementType(l)(elemType, options, "NORMAL") {
			m.EncodeNormal(l, tenant, options)(w, elemType)
			return
		}

		if isMovementType(l)(elemType, options, "TELEPORT") {
			m.EncodeTeleport(l, tenant, options)(w, elemType)
			return
		}

		if isMovementType(l)(elemType, options, "START_FALL_DOWN") {
			m.EncodeStartFallDown(l, tenant, options)(w, elemType)
			return
		}
		if isMovementType(l)(elemType, options, "FLYING_BLOCK") {
			m.EncodeFlyingBlock(l, tenant, options)(w, elemType)
			return
		}
		if isMovementType(l)(elemType, options, "JUMP") {
			m.EncodeJump(l, tenant, options)(w, elemType)
			return
		}
		if isMovementType(l)(elemType, options, "STAT_CHANGE") {
			m.EncodeStatChange(l, tenant, options)(w, elemType)
			return
		}
		w.WriteByte(m.BMoveAction)
		w.WriteInt16(m.TElapse)
	}
}

func (m *Element) EncodeNormal(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer, elemType byte) {
	return func(w *response.Writer, elemType byte) {
		w.WriteInt16(m.X)
		w.WriteInt16(m.Y)
		w.WriteInt16(m.Vx)
		w.WriteInt16(m.Vy)
		w.WriteInt16(m.Fh)
		if isMovementName(l)(elemType, options, "FALL_DOWN") {
			w.WriteInt16(m.FhFallStart)
		}
		if tenant.Region != "GMS" || tenant.MajorVersion > 87 {
			w.WriteInt16(m.XOffset)
			w.WriteInt16(m.YOffset)
		}
		w.WriteByte(m.BMoveAction)
		w.WriteInt16(m.TElapse)
	}
}

func (m *Element) EncodeTeleport(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer, elemType byte) {
	return func(w *response.Writer, elemType byte) {
		w.WriteInt16(m.X)
		w.WriteInt16(m.Y)
		w.WriteInt16(m.Fh)
		w.WriteByte(m.BMoveAction)
		w.WriteInt16(m.TElapse)
	}
}

func (m *Element) EncodeStartFallDown(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer, elemType byte) {
	return func(w *response.Writer, elemType byte) {
		w.WriteInt16(m.Vx)
		w.WriteInt16(m.Vy)
		w.WriteInt16(m.FhFallStart)
		w.WriteByte(m.BMoveAction)
		w.WriteInt16(m.TElapse)
	}
}

func (m *Element) EncodeFlyingBlock(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer, elemType byte) {
	return func(w *response.Writer, elemType byte) {
		w.WriteInt16(m.X)
		w.WriteInt16(m.Y)
		w.WriteInt16(m.Vx)
		w.WriteInt16(m.Vy)
		w.WriteByte(m.BMoveAction)
		w.WriteInt16(m.TElapse)
	}
}

func (m *Element) EncodeJump(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer, elemType byte) {
	return func(w *response.Writer, elemType byte) {
		w.WriteInt16(m.Vx)
		w.WriteInt16(m.Vy)
		w.WriteByte(m.BMoveAction)
		w.WriteInt16(m.TElapse)
	}
}

func (m *Element) EncodeStatChange(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer, elemType byte) {
	return func(w *response.Writer, elemType byte) {
		w.WriteByte(m.BStat)
	}
}

func (m *Element) Decode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {

		var elemType = r.ReadByte()
		m.ElemType = elemType
		if isMovementType(l)(elemType, options, "NORMAL") {
			m.DecodeNormal(l, tenant, options)(r, elemType)
			return
		}
		if isMovementType(l)(elemType, options, "TELEPORT") {
			m.DecodeTeleport(l, tenant, options)(r, elemType)
			return
		}
		if isMovementType(l)(elemType, options, "START_FALL_DOWN") {
			m.DecodeStartFallDown(l, tenant, options)(r, elemType)
			return
		}
		if isMovementType(l)(elemType, options, "FLYING_BLOCK") {
			m.DecodeFlyingBlock(l, tenant, options)(r, elemType)
			return
		}
		if isMovementType(l)(elemType, options, "JUMP") {
			m.DecodeJump(l, tenant, options)(r, elemType)
			return
		}
		if isMovementType(l)(elemType, options, "STAT_CHANGE") {
			m.DecodeStatChange(l, tenant, options)(r, elemType)
			return
		}
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	}
}

func (m *Element) DecodeNormal(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader, elemType byte) {
	return func(r *request.Reader, elemType byte) {
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.Fh = r.ReadInt16()
		if isMovementName(l)(elemType, options, "FALL_DOWN") {
			m.FhFallStart = r.ReadInt16()
		}
		if tenant.Region != "GMS" || tenant.MajorVersion > 83 {
			m.XOffset = r.ReadInt16()
			m.YOffset = r.ReadInt16()
		}
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	}
}

func (m *Element) DecodeTeleport(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader, elemType byte) {
	return func(r *request.Reader, elemType byte) {
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Fh = r.ReadInt16()
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	}
}

func (m *Element) DecodeStartFallDown(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader, elemType byte) {
	return func(r *request.Reader, elemType byte) {
		m.X = m.StartX
		m.Y = m.StartY
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.FhFallStart = r.ReadInt16()
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	}
}

func (m *Element) DecodeFlyingBlock(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader, elemType byte) {
	return func(r *request.Reader, elemType byte) {
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	}
}

func (m *Element) DecodeJump(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader, elemType byte) {
	return func(r *request.Reader, elemType byte) {
		m.X = m.StartX
		m.Y = m.StartY
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	}
}

func (m *Element) DecodeStatChange(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader, elemType byte) {
	return func(r *request.Reader, elemType byte) {
		m.BStat = r.ReadByte()
	}
}

func movementPathAttrFromOptions(l logrus.FieldLogger) func(attr byte, options map[string]interface{}) (string, string) {
	return func(attr byte, options map[string]interface{}) (string, string) {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["types"]; !ok {
			l.Errorf("Code [%d] not configured. Defaulting to 99 which will likely cause a client crash.", attr)
			return "NOT_FOUND", "DEFAULT"
		}

		var codes []interface{}
		if codes, ok = genericCodes.([]interface{}); !ok {
			l.Errorf("Code [%d] not configured. Defaulting to 99 which will likely cause a client crash.", attr)
			return "NOT_FOUND", "DEFAULT"
		}

		if len(codes) == 0 || attr < 0 || attr >= byte(len(codes)) {
			l.Errorf("Code [%d] not configured. Defaulting to 99 which will likely cause a client crash.", attr)
			return "NOT_FOUND", "DEFAULT"
		}

		var theType map[string]interface{}
		if theType, ok = codes[attr].(map[string]interface{}); !ok {
			l.Errorf("Code [%d] not configured. Defaulting to 99 which will likely cause a client crash.", attr)
			return "NOT_FOUND", "DEFAULT"
		}

		return theType["Name"].(string), theType["Type"].(string)
	}
}

func isMovementType(l logrus.FieldLogger) func(reference byte, options map[string]interface{}, movementType string) bool {
	return func(reference byte, options map[string]interface{}, movementType string) bool {
		_, t := movementPathAttrFromOptions(l)(reference, options)
		return t == movementType
	}
}

func isMovementName(l logrus.FieldLogger) func(reference byte, options map[string]interface{}, movementName string) bool {
	return func(reference byte, options map[string]interface{}, movementName string) bool {
		n, _ := movementPathAttrFromOptions(l)(reference, options)
		return n == movementName
	}
}
