package model

import (
	"github.com/Chronicle20/atlas-socket/response"
)

func WritePaddedString(w *response.Writer, str string, number int) {
	if len(str) > number {
		w.WriteByteArray([]byte(str)[:number])
	} else {
		w.WriteByteArray([]byte(str))
		w.WriteByteArray(make([]byte, number-len(str)))
	}
}
