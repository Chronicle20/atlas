package reactor

import (
	"strconv"
)

// ReactorRestModel represents a reactor from the atlas-reactors REST API
type ReactorRestModel struct {
	Id   uint32 `json:"-"`
	Name string `json:"name"`
}

func (r ReactorRestModel) GetName() string {
	return "reactors"
}

func (r ReactorRestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

func (r *ReactorRestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// ExtractReactor is a pass-through extractor for ReactorRestModel
func ExtractReactor(r ReactorRestModel) (ReactorRestModel, error) {
	return r, nil
}

// ResetReactorsInputRestModel is the body of POST .../reactors/reset.
// MinState is a pointer so "reset every reactor" (nil) and "reset only
// reactors at state 0" (pointer to 0) are distinguishable.
type ResetReactorsInputRestModel struct {
	Id       string `json:"-"`
	MinState *int8  `json:"minState,omitempty"`
}

func (r ResetReactorsInputRestModel) GetName() string {
	return "reactors"
}

func (r ResetReactorsInputRestModel) GetID() string {
	return r.Id
}

// ShuffleReactorsInputRestModel is the (empty) body of POST .../reactors/shuffle.
type ShuffleReactorsInputRestModel struct {
	Id string `json:"-"`
}

func (r ShuffleReactorsInputRestModel) GetName() string {
	return "reactors"
}

func (r ShuffleReactorsInputRestModel) GetID() string {
	return r.Id
}
