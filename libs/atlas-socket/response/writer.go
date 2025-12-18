package response

import (
	"bytes"
	"encoding/binary"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"io"
	"strings"
)

type Writer struct {
	l logrus.FieldLogger
	o *bytes.Buffer
}

//goland:noinspection GoUnusedExportedFunction
func NewWriter(l logrus.FieldLogger) *Writer {
	return &Writer{l, new(bytes.Buffer)}
}

// WriteInt8 -
func (w *Writer) WriteInt8(data int8) { w.WriteByte(uint8(data)) }

// WriteInt16 -
func (w *Writer) WriteInt16(data int16) { w.WriteShort(uint16(data)) }

// WriteInt32 -
func (w *Writer) WriteInt32(data int32) { w.WriteInt(uint32(data)) }

// WriteInt64 -
func (w *Writer) WriteInt64(data int64) { w.WriteLong(uint64(data)) }

func (w *Writer) WriteInt(val uint32) {
	err := binary.Write(w.o, binary.LittleEndian, val)
	if err != nil {
		w.l.WithError(err).Fatal("Writing int value")
	}
}

func (w *Writer) WriteShort(val uint16) {
	err := binary.Write(w.o, binary.LittleEndian, val)
	if err != nil {
		w.l.WithError(err).Fatal("Writing short value")
	}
}

func (w *Writer) WriteLong(val uint64) {
	err := binary.Write(w.o, binary.LittleEndian, val)
	if err != nil {
		w.l.WithError(err).Fatal("Writing long value")
	}
}

//goland:noinspection GoStandardMethods
func (w *Writer) WriteByte(val byte) {
	err := binary.Write(w.o, binary.LittleEndian, val)
	if err != nil {
		w.l.WithError(err).Fatal("Writing byte value")
	}
}

func (w *Writer) WriteByteArray(bytes []byte) {
	for i := 0; i < len(bytes); i++ {
		err := binary.Write(w.o, binary.LittleEndian, bytes[i])
		if err != nil {
			w.l.WithError(err).Fatal("Writing byte value")
		}
	}
}

func (w *Writer) WriteBool(val bool) {
	i := 1
	if !val {
		i = 0
	}
	w.WriteByte(byte(i))
}

func (w *Writer) WriteAsciiString(s string) {

	e := japanese.ShiftJIS.NewEncoder()
	r := strings.NewReader(s)
	ebs, err := io.ReadAll(transform.NewReader(r, e))
	if err != nil {
		w.WriteShort(uint16(len(s)))
		w.WriteByteArray([]byte(s))
		return
	}
	w.WriteShort(uint16(len(ebs)))
	w.WriteByteArray(ebs)
}

func (w *Writer) WriteKeyValue(key byte, value uint32) {
	w.WriteByte(key)
	w.WriteInt(value)
}

func (w *Writer) Bytes() []byte {
	return w.o.Bytes()
}

func (w *Writer) Skip(amount int) {
	ba := make([]byte, 0)
	for i := 0; i < amount; i++ {
		ba = append(ba, 0)
	}
	w.WriteByteArray(ba)
}
