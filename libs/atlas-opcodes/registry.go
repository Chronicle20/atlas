package opcodes

import (
	"strconv"
)

// Registry provides bidirectional lookups between operation names and opcodes.
type Registry struct {
	writersByName   map[string]entry
	writersByOpCode map[uint16]entry
	handlersByName  map[string]entry
	handlersByOp    map[uint16]entry
}

type entry struct {
	opCode  uint16
	name    string
	options map[string]interface{}
}

// NewRegistry builds a Registry from handler and writer configuration.
func NewRegistry(handlers []HandlerConfig, writers []WriterConfig) Registry {
	r := Registry{
		writersByName:   make(map[string]entry),
		writersByOpCode: make(map[uint16]entry),
		handlersByName:  make(map[string]entry),
		handlersByOp:    make(map[uint16]entry),
	}

	for _, wc := range writers {
		op, err := strconv.ParseUint(wc.OpCode, 0, 16)
		if err != nil {
			continue
		}
		e := entry{opCode: uint16(op), name: wc.Writer, options: wc.Options}
		r.writersByName[wc.Writer] = e
		r.writersByOpCode[uint16(op)] = e
	}

	for _, hc := range handlers {
		op, err := strconv.ParseUint(hc.OpCode, 0, 16)
		if err != nil {
			continue
		}
		e := entry{opCode: uint16(op), name: hc.Handler, options: hc.Options}
		r.handlersByName[hc.Handler] = e
		r.handlersByOp[uint16(op)] = e
	}

	return r
}

// WriterOpCode returns the opcode for a writer name.
func (r Registry) WriterOpCode(name string) (uint16, bool) {
	e, ok := r.writersByName[name]
	if !ok {
		return 0, false
	}
	return e.opCode, true
}

// WriterName returns the writer name for an opcode.
func (r Registry) WriterName(opCode uint16) (string, bool) {
	e, ok := r.writersByOpCode[opCode]
	if !ok {
		return "", false
	}
	return e.name, true
}

// WriterOptions returns the options map for a writer name.
func (r Registry) WriterOptions(name string) map[string]interface{} {
	e, ok := r.writersByName[name]
	if !ok {
		return nil
	}
	return e.options
}

// HandlerOpCode returns the opcode for a handler name.
func (r Registry) HandlerOpCode(name string) (uint16, bool) {
	e, ok := r.handlersByName[name]
	if !ok {
		return 0, false
	}
	return e.opCode, true
}

// HandlerName returns the handler name for an opcode.
func (r Registry) HandlerName(opCode uint16) (string, bool) {
	e, ok := r.handlersByOp[opCode]
	if !ok {
		return "", false
	}
	return e.name, true
}

// HandlerOptions returns the options map for a handler name.
func (r Registry) HandlerOptions(name string) map[string]interface{} {
	e, ok := r.handlersByName[name]
	if !ok {
		return nil
	}
	return e.options
}

// Writers returns all writer configurations as a slice.
func (r Registry) Writers() []WriterConfig {
	result := make([]WriterConfig, 0, len(r.writersByName))
	for _, e := range r.writersByName {
		result = append(result, WriterConfig{
			OpCode:  "0x" + strconv.FormatUint(uint64(e.opCode), 16),
			Writer:  e.name,
			Options: e.options,
		})
	}
	return result
}

// Handlers returns all handler configurations as a slice.
func (r Registry) Handlers() []HandlerConfig {
	result := make([]HandlerConfig, 0, len(r.handlersByName))
	for _, e := range r.handlersByName {
		result = append(result, HandlerConfig{
			OpCode:  "0x" + strconv.FormatUint(uint64(e.opCode), 16),
			Handler: e.name,
			Options: e.options,
		})
	}
	return result
}
