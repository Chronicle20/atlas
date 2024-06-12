package response

import (
	"encoding/json"
)

// objectMap is a simple representation of a json map. key is a string, and value is a nested object
type objectMap map[string]interface{}

// ObjectMapper maps a objectMap to a concrete data type
type ObjectMapper func(objectMap) interface{}

// ConditionalMapperProvider returns a string representing the type of object the ObjectMapper handles, as well as the
// ObjectMapper itself.
type ConditionalMapperProvider func() (string, ObjectMapper)

// concreteObjectProvider returns a new (empty) concrete object
type concreteObjectProvider func() interface{}

// DataSegment is a slice of concrete objects
type DataSegment []interface{}

// UnmarshalRoot will take a raw byte array, and using mapper functions, produce a data DataSegment and includes
// DataSegment representing a jsonapi.org request response
func UnmarshalRoot(data []byte, root ObjectMapper, options ...ConditionalMapperProvider) (DataSegment, DataSegment, error) {
	var dataResult DataSegment
	var includeResult DataSegment

	var single = struct {
		Data     objectMap
		Included []objectMap
	}{}
	err := json.Unmarshal(data, &single)
	if err == nil {
		dataResult = produceSingle(single.Data, root)
		includeResult = produceList(single.Included, includeMapper(options...))
		return dataResult, includeResult, nil
	} else {
		var list = struct {
			Data     []objectMap
			Included []objectMap
		}{}
		err = json.Unmarshal(data, &list)
		if err == nil {
			dataResult = produceList(list.Data, root)
			includeResult = produceList(list.Included, includeMapper(options...))
			return dataResult, includeResult, nil
		}
		return nil, nil, err
	}
}

// includeMapper represents a ObjectMapper for handling the jsonapi.org includes data section
func includeMapper(options ...ConditionalMapperProvider) ObjectMapper {
	return func(o objectMap) interface{} {
		return addInclude(o, options...)
	}
}

// produceSingle will produce a DataSegment given a single objectMap from a ObjectMapper
func produceSingle(o objectMap, m ObjectMapper) DataSegment {
	return append(make([]interface{}, 0), m(o))
}

// produceList will produce a DataSegment given a objectMap slice from a ObjectMapper
func produceList(o []objectMap, m ObjectMapper) DataSegment {
	if len(o) > 0 {
		var result = make([]interface{}, 0)
		for _, x := range o {
			result = append(result, m(x))
		}
		return result
	}
	return nil
}

// transformMap will populate the concrete object provided by the concreteObjectProvider from the objectMap and return it.
func transformMap(dpf concreteObjectProvider, x objectMap) interface{} {
	b, err := json.Marshal(x)
	if err == nil {
		r := dpf()
		err = json.Unmarshal(b, r)
		if err == nil {
			return r
		}
	}
	return nil
}

func MapperFunc(dpf concreteObjectProvider) ObjectMapper {
	return func(x objectMap) interface{} {
		return transformMap(dpf, x)
	}
}

func UnmarshalData(tf string, dpf concreteObjectProvider) (string, ObjectMapper) {
	return tf, MapperFunc(dpf)
}

// addInclude processes a map structure representing the jsonapi.org data object to produce a concrete struct.
// given a set of functions which could produce a concrete struct
func addInclude(x objectMap, options ...ConditionalMapperProvider) interface{} {
	t := x["type"].(string)
	for _, o := range options {
		tf, mf := o()
		if t == tf {
			return mf(x)
		}
	}
	return nil
}
