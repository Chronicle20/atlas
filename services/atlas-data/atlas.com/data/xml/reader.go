package xml

import (
	"encoding/xml"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"io"
	"os"
)

// deprecated
func Read(path string) (*Node, error) {
	n, err := FromPathProvider(path)()
	if err != nil {
		return nil, err
	}
	return &n, nil
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
