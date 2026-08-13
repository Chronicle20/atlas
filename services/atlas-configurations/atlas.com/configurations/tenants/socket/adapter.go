package socket

import (
	configsocket "atlas-configurations/socket"
)

// ToValidationInput flattens this tree's RestModel into the shared validator's
// neutral Input. It is ~20 lines of mechanical copying that exist so the ~150
// lines of rules and their tests do not have to be duplicated per tree.
//
// Writers carry no validator, so their Binding.Validator is left empty; the
// shared validator only requires one for handlers.
func ToValidationInput(rm RestModel) configsocket.Input {
	in := configsocket.Input{
		Handlers:            make([]configsocket.Binding, 0, len(rm.Handlers)),
		Writers:             make([]configsocket.Binding, 0, len(rm.Writers)),
		UnsupportedHandlers: rm.Unsupported.Handlers,
		UnsupportedWriters:  rm.Unsupported.Writers,
	}
	for _, h := range rm.Handlers {
		in.Handlers = append(in.Handlers, configsocket.Binding{
			Name:      h.Handler,
			OpCode:    h.OpCode,
			Validator: h.Validator,
			Services:  h.Services,
		})
	}
	for _, w := range rm.Writers {
		in.Writers = append(in.Writers, configsocket.Binding{
			Name:     w.Writer,
			OpCode:   w.OpCode,
			Services: w.Services,
		})
	}
	return in
}
