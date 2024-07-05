package xml

import (
	"encoding/xml"
	"errors"
	"io/ioutil"
	"os"
)

func Read(path string) (*Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return nil, errors.New("not a valid xml file")
	}

	byteValue, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var equipment Node
	err = xml.Unmarshal(byteValue, &equipment)
	if err != nil {
		return nil, err
	}

	return &equipment, nil
}
