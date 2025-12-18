package xml

import (
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"io"
	"os"
	"path/filepath"
)

// deprecated
func Read(path string) (*Node, error) {
	n, err := FromPathProvider(path)()
	if err != nil {
		return nil, err
	}
	return &n, nil
}

type IdProvider func(path string, id uint32) model.Provider[Node]

func PadUint32(value uint32, digits uint8) string {
	// Use %0Nd where N is the desired digit count
	format := fmt.Sprintf("%%0%dd", digits)
	return fmt.Sprintf(format, value)
}

func FromParentPathProvider(pad uint8) func(path string, id uint32) model.Provider[Node] {
	return func(path string, id uint32) model.Provider[Node] {
		return FromPathProvider(filepath.Join(path, fmt.Sprintf("%s.img.xml", PadUint32(id, pad))))
	}
}

func FromPathProvider(path string) model.Provider[Node] {
	f, err := os.Open(path)
	if err != nil {
		return model.ErrorProvider[Node](err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return model.ErrorProvider[Node](err)
	}

	if stat.IsDir() {
		return model.ErrorProvider[Node](errors.New("not a valid xml file"))
	}

	byteValue, err := io.ReadAll(f)
	if err != nil {
		return model.ErrorProvider[Node](err)
	}
	return FromByteArrayProvider(byteValue)
}

func FromByteArrayProvider(data []byte) model.Provider[Node] {
	var n Node
	err := xml.Unmarshal(data, &n)
	if err != nil {
		return model.ErrorProvider[Node](err)
	}
	return model.FixedProvider(n)
}
