package socket

type Configurator func(s *config)

//goland:noinspection GoUnusedExportedFunction
func SetIpAddress(ipAddress string) func(*config) {
	return func(s *config) {
		s.ipAddress = ipAddress
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetPort(port int) func(*config) {
	return func(s *config) {
		s.port = port
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetCreator(creator Creator) Configurator {
	return func(s *config) {
		s.creator = creator
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetDestroyer(destroyer Destroyer) Configurator {
	return func(s *config) {
		s.destroyer = destroyer
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetMessageDecryptor(decryptor MessageDecryptor) Configurator {
	return func(s *config) {
		s.decryptor = decryptor
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetReadWriter(rw OpReadWriter) Configurator {
	return func(s *config) {
		s.rw = rw
	}
}

func SetHandlers(producer HandlerProducer) Configurator {
	return func(s *config) {
		s.handlers = producer()
	}
}
