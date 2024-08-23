package model

import (
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Movement struct {
	StartX   int16
	StartY   int16
	Elements []EncoderDecoder
}

func (m *Movement) Decode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		m.StartX = r.ReadInt16()
		m.StartY = r.ReadInt16()

		numElems := r.ReadByte()
		var elems = make([]EncoderDecoder, numElems)
		for i := byte(0); i < numElems; i++ {
			var elem EncoderDecoder
			var elemType = r.ReadByte()

			if isMovementType(l)(elemType, options, "NORMAL") {
				elem = &NormalElement{Element{ElemType: elemType, StartX: m.StartX, StartY: m.StartY}}
			} else if isMovementType(l)(elemType, options, "TELEPORT") {
				elem = &TeleportElement{Element{ElemType: elemType, StartX: m.StartX, StartY: m.StartY}}
			} else if isMovementType(l)(elemType, options, "START_FALL_DOWN") {
				elem = &StartFallDownElement{Element{ElemType: elemType, StartX: m.StartX, StartY: m.StartY}}
			} else if isMovementType(l)(elemType, options, "FLYING_BLOCK") {
				elem = &FlyingBlockElement{Element{ElemType: elemType, StartX: m.StartX, StartY: m.StartY}}
			} else if isMovementType(l)(elemType, options, "JUMP") {
				elem = &JumpElement{Element{ElemType: elemType, StartX: m.StartX, StartY: m.StartY}}
			} else if isMovementType(l)(elemType, options, "STAT_CHANGE") {
				elem = &StatChangeElement{Element{ElemType: elemType, StartX: m.StartX, StartY: m.StartY}}
			} else {
				elem = &Element{ElemType: elemType}
			}
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

func (m *Element) Decode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	}
}

func (m *Element) EncodeType(w *response.Writer) {
	w.WriteByte(m.ElemType)
}

type NormalElement struct {
	Element
}

type TeleportElement struct {
	Element
}

type StartFallDownElement struct {
	Element
}

type FlyingBlockElement struct {
	Element
}

type JumpElement struct {
	Element
}

type StatChangeElement struct {
	Element
}

func (m *NormalElement) Decode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.Fh = r.ReadInt16()
		if isMovementName(l)(m.ElemType, options, "FALL_DOWN") {
			m.FhFallStart = r.ReadInt16()
		}
		if tenant.Region() != "GMS" || tenant.MajorVersion() > 83 {
			m.XOffset = r.ReadInt16()
			m.YOffset = r.ReadInt16()
		}
		m.Element.Decode(l, tenant, options)(r)
	}
}

func (m *TeleportElement) Decode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Fh = r.ReadInt16()
		m.Element.Decode(l, tenant, options)(r)
	}
}

func (m *StartFallDownElement) Decode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		m.X = m.StartX
		m.Y = m.StartY
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.FhFallStart = r.ReadInt16()
		m.Element.Decode(l, tenant, options)(r)
	}
}

func (m *FlyingBlockElement) Decode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.Element.Decode(l, tenant, options)(r)
	}
}

func (m *JumpElement) Decode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		m.X = m.StartX
		m.Y = m.StartY
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.Element.Decode(l, tenant, options)(r)
	}
}

func (m *StatChangeElement) Decode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		m.BStat = r.ReadByte()
	}
}

func (m *Movement) Encode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(m.StartX)
		w.WriteInt16(m.StartY)
		w.WriteByte(byte(len(m.Elements)))
		for _, element := range m.Elements {
			element.EncodeType(w)
			element.Encode(l, tenant, options)(w)
		}
	}
}

func (m *Element) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteByte(m.BMoveAction)
		w.WriteInt16(m.TElapse)
	}
}

func (m *NormalElement) Encode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(m.X)
		w.WriteInt16(m.Y)
		w.WriteInt16(m.Vx)
		w.WriteInt16(m.Vy)
		w.WriteInt16(m.Fh)
		if isMovementName(l)(m.ElemType, options, "FALL_DOWN") {
			w.WriteInt16(m.FhFallStart)
		}
		if tenant.Region() != "GMS" || tenant.MajorVersion() > 87 {
			w.WriteInt16(m.XOffset)
			w.WriteInt16(m.YOffset)
		}
		m.Element.Encode(l, tenant, options)(w)
	}
}

func (m *TeleportElement) Encode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(m.X)
		w.WriteInt16(m.Y)
		w.WriteInt16(m.Fh)
		m.Element.Encode(l, tenant, options)(w)
	}
}

func (m *StartFallDownElement) Encode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(m.Vx)
		w.WriteInt16(m.Vy)
		w.WriteInt16(m.FhFallStart)
		m.Element.Encode(l, tenant, options)(w)
	}
}

func (m *FlyingBlockElement) Encode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(m.X)
		w.WriteInt16(m.Y)
		w.WriteInt16(m.Vx)
		w.WriteInt16(m.Vy)
		m.Element.Encode(l, tenant, options)(w)
	}
}

func (m *JumpElement) Encode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(m.Vx)
		w.WriteInt16(m.Vy)
		m.Element.Encode(l, tenant, options)(w)
	}
}

func (m *StatChangeElement) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteByte(m.BStat)
	}
}

func movementPathAttrFromOptions(l logrus.FieldLogger) func(attr byte, options map[string]interface{}) (string, string) {
	return func(attr byte, options map[string]interface{}) (string, string) {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["types"]; !ok {
			l.Errorf("Code [%d] not configured for use in movement. Defaulting to 99 which will likely cause a client crash.", attr)
			return "NOT_FOUND", "DEFAULT"
		}

		var codes []interface{}
		if codes, ok = genericCodes.([]interface{}); !ok {
			l.Errorf("Code [%d] not configured for use in movement. Defaulting to 99 which will likely cause a client crash.", attr)
			return "NOT_FOUND", "DEFAULT"
		}

		if len(codes) == 0 || attr < 0 || attr >= byte(len(codes)) {
			l.Errorf("Code [%d] not configured for use in movement. Defaulting to 99 which will likely cause a client crash.", attr)
			return "NOT_FOUND", "DEFAULT"
		}

		var theType map[string]interface{}
		if theType, ok = codes[attr].(map[string]interface{}); !ok {
			l.Errorf("Code [%d] not configured for use in movement. Defaulting to 99 which will likely cause a client crash.", attr)
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
